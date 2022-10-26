// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	lserrors "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

// Scope contains the visible installations of a specific installation.
// This context is later used to validate and get import data
type Scope struct {
	// Name is the name of the current installation's context.
	// By default, it is the source name of the parent.
	Name string
	// Parent is the installation is encompassed in.
	// Parents are handled separately as installation have access to the same imports as their parent.
	Parent *InstallationAndImports
	// Siblings are installations with the same parent.
	// The installation has access to the exports of these components
	Siblings []*InstallationAndImports
	// External describes the external installation context that contains
	// context specific configuration.
	External ExternalContext
}

// SetInstallationContext determines the current context and updates the operation context.
func (o *Operation) SetInstallationContext(ctx context.Context) error {
	newCtx, err := GetInstallationContext(ctx, o.Client(), o.Inst.GetInstallation())
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
	inst *lsv1alpha1.Installation) (*Scope, error) {
	parentInst, siblingInstallations, err := GetParentAndSiblings(ctx, kubeClient, inst)
	if err != nil {
		return nil, err
	}

	externalCtx, err := GetExternalContext(ctx, kubeClient, inst)
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

	return &Scope{
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
	// Overwriter is the component version overwriter used for this installation.
	Overwriter componentoverwrites.Overwriter
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
		filter := func(a lsv1alpha1.Installation) bool {
			return a.Name == inst.Name
		}
		rootInstallations, err := GetRootInstallations(ctx, kubeClient, filter, client.InNamespace(inst.Namespace))
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
func GetExternalContext(ctx context.Context, kubeClient client.Client, inst *lsv1alpha1.Installation) (ExternalContext, error) {
	logger, ctx := logging.FromContextOrNew(ctx, nil)
	lsCtx := &lsv1alpha1.Context{}
	var overwriter componentoverwrites.Overwriter
	var cvo *lsv1alpha1.ComponentVersionOverwrites
	if len(inst.Spec.Context) != 0 {
		if err := kubeClient.Get(ctx, kutil.ObjectKey(inst.Spec.Context, inst.Namespace), lsCtx); err != nil {
			return ExternalContext{}, lserrors.NewWrappedError(err,
				"Context", "GetContext", err.Error())
		}

		// check for ComponentVersionOverwrites
		if len(lsCtx.ComponentVersionOverwritesReference) > 0 {
			cvo = &lsv1alpha1.ComponentVersionOverwrites{}
			if err := kubeClient.Get(ctx, kutil.ObjectKey(lsCtx.ComponentVersionOverwritesReference, inst.Namespace), cvo); err != nil {
				if apierrors.IsNotFound(err) {
					return ExternalContext{}, lserrors.NewWrappedError(err, "ComponentVersionOverwrites", "GetComponentVersionOverwrites", fmt.Sprintf("context '%s' references ComponentVersionOverwrites resource '%s', which cannot be found: %s", lsCtx.Name, lsCtx.ComponentVersionOverwritesReference, err.Error()))
				}
				return ExternalContext{}, lserrors.NewWrappedError(err, "ComponentVersionOverwrites", "GetComponentVersionOverwrites", err.Error())
			}
		}
	}

	if cvo != nil {
		overwriter = componentoverwrites.NewSubstitutions(cvo.Overwrites)
		logger.Debug("Found ComponentVersionOverwrites for context", "context", inst.Spec.Context, lc.KeyResource, lsCtx.ComponentVersionOverwritesReference, lc.KeyResourceKind, "ComponentVersionOverwrites")
	}

	cdRef := GetReferenceFromComponentDescriptorDefinition(inst.Spec.ComponentDescriptor)
	if cdRef == nil {
		// no component descriptor is configured
		return ExternalContext{
			Context:    *lsCtx,
			Overwriter: overwriter,
		}, nil
	}

	cond, err := ApplyComponentOverwrite(ctx, inst, overwriter, lsCtx, cdRef)
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
		Overwriter:       overwriter,
	}, nil
}

// ApplyComponentOverwrite applies a component overwrite for the component reference if applicable.
// The overwriter can be nil
func ApplyComponentOverwrite(ctx context.Context, inst *lsv1alpha1.Installation, overwriter componentoverwrites.Overwriter, lsCtx *lsv1alpha1.Context, cdRef *lsv1alpha1.ComponentDescriptorReference) (*lsv1alpha1.Condition, error) {
	logger, _ := logging.FromContextOrNew(ctx, nil)
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

	overwritten := overwriter.Replace(cdRef)
	if overwritten {
		diff := componentoverwrites.ReferenceDiff(oldRef, cdRef)
		logger.Info("Component reference has been overwritten",
			"repositoryContext", diff.OverwriteToString(componentoverwrites.RepoCtx, true),
			"componentName", diff.OverwriteToString(componentoverwrites.ComponentName, true),
			"version", diff.OverwriteToString(componentoverwrites.Version, true))
		cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue,
			"FoundOverwrite",
			diff.String())
		return &cond, nil
	}

	cond = lsv1alpha1helper.UpdatedCondition(cond,
		lsv1alpha1.ConditionFalse,
		"No overwrite defined",
		"component reference has not been overwritten")
	return &cond, nil
}
