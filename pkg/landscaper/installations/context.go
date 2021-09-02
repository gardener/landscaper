// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// Context defines the internal context with additional enhanced context information.
type Context struct {
	lsv1alpha1.Context
	// ComponentName defines the unique name of the component containing the resource.
	ComponentName string
	// ComponentVersion defines the version of the component.
	ComponentVersion string
}

// ComponentDescriptorRef returns the component descriptor reference for the current installation
func (c *Context) ComponentDescriptorRef() *lsv1alpha1.ComponentDescriptorReference {
	ref := &lsv1alpha1.ComponentDescriptorReference{}
	ref.RepositoryContext = c.RepositoryContext
	ref.ComponentName = c.ComponentName
	ref.Version = c.ComponentVersion
	return ref
}

// GetContext resolves the context for an installation and applies defaults or overwrites if applicable.
func GetContext(ctx context.Context, kubeClient client.Client, inst *lsv1alpha1.Installation, overwriter componentoverwrites.Overwriter) (Context, error) {
	lsCtx := &lsv1alpha1.Context{}
	if len(inst.Spec.Context) != 0 {
		if err := kubeClient.Get(ctx, kutil.ObjectKey(inst.Spec.Context, inst.Namespace), lsCtx); err != nil {
			return Context{}, lserrors.NewWrappedError(err,
				"Context", "GetContext", err.Error())
		}
	}

	cdRef := GetReferenceFromComponentDescriptorDefinition(inst.Spec.ComponentDescriptor)

	cond, err := ApplyComponentOverwrite(overwriter, lsCtx, cdRef)
	if err != nil {
		return Context{}, lserrors.NewWrappedError(err,
			"Context", "OverwriteComponentReference", err.Error())
	}
	if cond != nil {
		inst.Status.Conditions = lsv1alpha1helper.MergeConditions(inst.Status.Conditions, *cond)
	}
	lsCtx.RepositoryContext = cdRef.RepositoryContext
	return Context{
		Context:          *lsCtx,
		ComponentName:    cdRef.ComponentName,
		ComponentVersion: cdRef.Version,
	}, nil
}

// ApplyComponentOverwrite applies a component overwrite for the component reference if applicable.
func ApplyComponentOverwrite(overwriter componentoverwrites.Overwriter, lsCtx *lsv1alpha1.Context, cdRef *lsv1alpha1.ComponentDescriptorReference) (*lsv1alpha1.Condition, error) {
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

// ContextFromInstallation returns the Context from an installation without remote context object.
// For real use cases use GetContext instead.
func ContextFromInstallation(inst *lsv1alpha1.Installation) Context {
	cdRef := GetReferenceFromComponentDescriptorDefinition(inst.Spec.ComponentDescriptor)
	if cdRef == nil {
		return Context{}
	}
	return Context{
		Context: lsv1alpha1.Context{
			RepositoryContext: cdRef.RepositoryContext,
		},
		ComponentName:    cdRef.ComponentName,
		ComponentVersion: cdRef.Version,
	}
}
