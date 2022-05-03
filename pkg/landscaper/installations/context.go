// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"errors"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lserrors "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

// Context contains the visible installations of a specific installation.
// This context is later used to validate and get import data
type Context struct {
	// Name is the name of the current installation's context.
	// By default, it is the source name of the parent.
	Name string
	// Parent is the installation is encompassed in.
	// Parents are handled separately as installation have access to the same imports as their parent.
	Parent *InstallationBase
	// Siblings are installations with the same parent.
	// The installation has access to the exports of these components
	Siblings []*InstallationBase
	// External describes the external installation context that contains
	// context specific configuration.
	External ExternalContext
}

// SetInstallationContext determines the current context and updates the operation context.
func (o *Operation) SetInstallationContext(ctx context.Context) error {
	newCtx, err := GetInstallationContext(ctx, o.Client(), o.Inst.Info, o.Overwriter)
	if err != nil {
		return err
	}
	o.context = *newCtx
	return nil
}

// GetInstallationContext determines the visible context of an installation.
// The visible context consists of the installation's parent and siblings.
// The context is later used to validate and get imported data.
func GetInstallationContext(ctx context.Context,
	kubeClient client.Client,
	inst *lsv1alpha1.Installation,
	overwriter componentoverwrites.Overwriter) (*Context, error) {
	parentInst, siblingInstallations, err := GetParentAndSiblings(ctx, kubeClient, inst)
	if err != nil {
		return nil, err
	}

	externalCtx, err := GetExternalContext(ctx, kubeClient, inst, overwriter)
	if err != nil {
		return nil, err
	}

	// set optional default repository context
	for _, inst := range siblingInstallations {
		if inst.Spec.ComponentDescriptor != nil &&
			inst.Spec.ComponentDescriptor.Reference != nil &&
			inst.Spec.ComponentDescriptor.Reference.RepositoryContext == nil {
			inst.Spec.ComponentDescriptor.Reference.RepositoryContext = externalCtx.RepositoryContext
		}
	}

	ctxName := ""
	if parentInst != nil {
		ctxName = lsv1alpha1helper.DataObjectSourceFromInstallation(parentInst)
	}

	return &Context{
		Name:   ctxName,
		Parent: CreateInternalInstallationBase(parentInst),
		// siblings are all encompassed installation of the parent installation
		Siblings: CreateInternalInstallationBases(siblingInstallations...),
		External: externalCtx,
	}, nil
}

// ExternalContext is the context defined by the external "Context" resource that
// is referenced by the installation.
// The external context contains additional parsed information.
// It should always be used to resolve the component descriptor of an installation.
type ExternalContext struct {
	lsv1alpha1.Context
	// ComponentName defines the unique name of the component containing the resource.
	ComponentName string
	// ComponentVersion defines the version of the component.
	ComponentVersion string
}

// ComponentDescriptorRef returns the component descriptor reference for the current installation
func (c *ExternalContext) ComponentDescriptorRef() *lsv1alpha1.ComponentDescriptorReference {
	if len(c.ComponentName) == 0 || len(c.ComponentVersion) == 0 {
		return nil
	}
	ref := &lsv1alpha1.ComponentDescriptorReference{}
	ref.RepositoryContext = c.RepositoryContext
	ref.ComponentName = c.ComponentName
	ref.Version = c.ComponentVersion
	return ref
}

// InjectComponentDescriptorRef injects the effective component descriptor ref into the given installation
func (c *ExternalContext) InjectComponentDescriptorRef(inst *lsv1alpha1.Installation) *lsv1alpha1.Installation {
	if inst.Spec.ComponentDescriptor != nil && inst.Spec.ComponentDescriptor.Inline != nil {
		// do not inject a different component reference for inlined defined component descriptors
		return inst
	}
	inst.Spec.ComponentDescriptor = &lsv1alpha1.ComponentDescriptorDefinition{
		Reference: c.ComponentDescriptorRef(),
	}
	return inst
}

// RegistryPullSecrets returns all registry pull secrets as list of object references.
func (c *ExternalContext) RegistryPullSecrets() []lsv1alpha1.ObjectReference {
	refs := make([]lsv1alpha1.ObjectReference, len(c.Context.RegistryPullSecrets))
	for i, r := range c.Context.RegistryPullSecrets {
		refs[i] = lsv1alpha1.ObjectReference{
			Name:      r.Name,
			Namespace: c.Context.Namespace,
		}
	}
	return refs
}

// GetParentAndSiblings determines the visible context of an installation.
// The visible context consists of the installation's parent and siblings.
// The context is later used to validate and get imported data.
func GetParentAndSiblings(ctx context.Context, kubeClient client.Client, inst *lsv1alpha1.Installation) (parent *lsv1alpha1.Installation, siblings []*lsv1alpha1.Installation, err error) {
	if IsRootInstallation(inst) {
		// get all root object as siblings
		rootInstallations, err := GetRootInstallations(ctx, kubeClient, func(a lsv1alpha1.Installation) bool {
			return a.Name == inst.Name
		}, client.InNamespace(inst.Namespace))
		if err != nil {
			return nil, nil, err
		}
		return nil, rootInstallations, err
	}

	// get the parent by owner reference
	parent, err = GetParent(ctx, kubeClient, inst)
	if err != nil {
		return nil, nil, err
	}

	// siblings are all encompassed installation of the parent installation
	siblings, err = ListSubinstallations(ctx, kubeClient, parent, func(found *lsv1alpha1.Installation) bool {
		return inst.Name == found.Name
	})
	if err != nil {
		return nil, nil, err
	}

	return parent, siblings, nil
}

// GetParent returns the parent of an installation.
// It returns nil if the installation has no parent
func GetParent(ctx context.Context, kubeClient client.Client, inst *lsv1alpha1.Installation) (*lsv1alpha1.Installation, error) {
	if IsRootInstallation(inst) {
		return nil, nil
	}
	// get the parent by owner reference
	parentName := GetParentInstallationName(inst)
	parent := &lsv1alpha1.Installation{}
	if err := read_write_layer.GetInstallation(ctx, kubeClient, client.ObjectKey{Name: parentName, Namespace: inst.Namespace}, parent); err != nil {
		return nil, err
	}
	return parent, nil
}

// GetInstallationContextName returns the name of the context of an installation.
// The context name is basically the name of the parent component.
func GetInstallationContextName(inst *lsv1alpha1.Installation) string {
	if IsRootInstallation(inst) {
		return ""
	}
	return lsv1alpha1helper.DataObjectSourceFromInstallationName(GetParentInstallationName(inst))
}

// IsRoot returns if the current component is a root component
func (o *Operation) IsRoot() bool {
	return o.Context().Parent == nil
}

// MissingRepositoryContextError defines a error when no repository context is defined.
var MissingRepositoryContextError = errors.New("RepositoryContextMissing")

// GetExternalContext resolves the context for an installation and applies defaults or overwrites if applicable.
func GetExternalContext(ctx context.Context, kubeClient client.Client, inst *lsv1alpha1.Installation, overwriter componentoverwrites.Overwriter) (ExternalContext, error) {
	lsCtx := &lsv1alpha1.Context{}
	if len(inst.Spec.Context) != 0 {
		if err := kubeClient.Get(ctx, kutil.ObjectKey(inst.Spec.Context, inst.Namespace), lsCtx); err != nil {
			return ExternalContext{}, lserrors.NewWrappedError(err,
				"Context", "GetContext", err.Error())
		}
	}

	cdRef := GetReferenceFromComponentDescriptorDefinition(inst.Spec.ComponentDescriptor)
	if cdRef == nil {
		// no component descriptor is configured
		return ExternalContext{
			Context: *lsCtx,
		}, nil
	}

	cond, err := ApplyComponentOverwrite(inst, overwriter, lsCtx, cdRef)
	if err != nil {
		return ExternalContext{}, lserrors.NewWrappedError(err,
			"Context", "OverwriteComponentReference", err.Error())
	}
	if cond != nil {
		inst.Status.Conditions = lsv1alpha1helper.MergeConditions(inst.Status.Conditions, *cond)
	}
	if cdRef.RepositoryContext == nil {
		return ExternalContext{}, MissingRepositoryContextError
	}
	lsCtx.RepositoryContext = cdRef.RepositoryContext
	return ExternalContext{
		Context:          *lsCtx,
		ComponentName:    cdRef.ComponentName,
		ComponentVersion: cdRef.Version,
	}, nil
}

// ApplyComponentOverwrite applies a component overwrite for the component reference if applicable.
// The overwriter can be nil
func ApplyComponentOverwrite(inst *lsv1alpha1.Installation, overwriter componentoverwrites.Overwriter,
	lsCtx *lsv1alpha1.Context, cdRef *lsv1alpha1.ComponentDescriptorReference) (*lsv1alpha1.Condition, error) {
	if cdRef == nil {
		return nil, nil
	}
	// default repository context if not defined
	if cdRef.RepositoryContext == nil {
		cdRef.RepositoryContext = lsCtx.RepositoryContext
	}

	if overwriter == nil {
		return nil, nil
	}

	cond := lsv1alpha1helper.InitCondition(lsv1alpha1.ComponentReferenceOverwriteCondition)
	if inst != nil {
		cond = lsv1alpha1helper.GetOrInitCondition(inst.Status.Conditions, lsv1alpha1.ComponentReferenceOverwriteCondition)
	}

	oldRef := cdRef.DeepCopy()

	overwritten, err := overwriter.Replace(cdRef)
	if err != nil {
		return nil, lserrors.NewWrappedError(err,
			"HandleComponentReference", "OverwriteComponentReference", err.Error())
	}
	if overwritten {
		diff := componentoverwrites.ReferenceDiff(oldRef, cdRef)
		cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue,
			"FoundOverwrite",
			diff)
		return &cond, nil
	}

	cond = lsv1alpha1helper.UpdatedCondition(cond,
		lsv1alpha1.ConditionFalse,
		"No overwrite defined",
		"component reference has not been overwritten")
	return &cond, nil
}
