// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
)

// Scope contains the visible installations of a specific installation.
// This scope is later used to validate and get import data
type Scope struct {
	// Name is the name of the current installation's context.
	// BY default it is the source name of the parent.
	Name string
	// Parent is the installation the installation is encompassed in.
	// Parents are handled separately as installation have access to the same imports as their parent.
	Parent *Installation

	// Siblings are installations with the same parent.
	// The installation has access to the exports of theses components
	Siblings []*InstallationBase
}

// SetInstallationScope determines the current context and updates the operation context.
func (o *Operation) SetInstallationScope(ctx context.Context) error {
	newCtx, err := o.DetermineInstallationScope(ctx)
	if err != nil {
		return err
	}
	o.scope = *newCtx
	return nil
}

// DetermineInstallationScope determines the visible context of an installation.
// The visible context consists of the installation's parent and siblings.
// The context is later used to validate and get imported data.
func (o *Operation) DetermineInstallationScope(ctx context.Context) (*Scope, error) {
	if IsRootInstallation(o.Inst.Info) {
		// get all root object as siblings
		rootInstallations, err := o.GetRootInstallations(ctx, func(inst lsv1alpha1.Installation) bool {
			return inst.Name == o.Inst.Info.Name
		}, client.InNamespace(o.Inst.Info.Namespace))
		if err != nil {
			return nil, err
		}
		return &Scope{Siblings: rootInstallations}, nil
	}

	// get the parent by owner reference
	parent, err := GetParent(ctx, o.Operation, &o.Inst.InstallationBase)
	if err != nil {
		return nil, err
	}

	// siblings are all encompassed installation of the parent installation
	subInstallations := make([]*lsv1alpha1.Installation, 0)
	for _, installationRef := range parent.Info.Status.InstallationReferences {
		if installationRef.Reference.Name == o.Inst.Info.Name {
			continue
		}
		subInst := &lsv1alpha1.Installation{}
		if err := o.Client().Get(ctx, installationRef.Reference.NamespacedName(), subInst); err != nil {
			return nil, err
		}
		subInstallations = append(subInstallations, subInst)
	}

	intSubInstallations, err := CreateInternalInstallationBases(ctx, o, subInstallations...)
	if err != nil {
		return nil, err
	}

	return &Scope{
		Name:     lsv1alpha1helper.DataObjectSourceFromInstallation(parent.Info),
		Parent:   parent,
		Siblings: intSubInstallations,
	}, nil
}

// GetParent returns the parent of a installation.
// It returns nil if the installation has no parent
func GetParent(ctx context.Context, op *operation.Operation, inst *InstallationBase) (*Installation, error) {
	if IsRootInstallation(inst.Info) {
		return nil, nil
	}
	// get the parent by owner reference
	parentName := GetParentInstallationName(inst.Info)
	parent := &lsv1alpha1.Installation{}
	if err := op.Client().Get(ctx, client.ObjectKey{Name: parentName, Namespace: inst.Info.Namespace}, parent); err != nil {
		return nil, err
	}
	parentCtx, err := GetContext(ctx, op.Client(), parent, op.ComponentsOverwriter())
	if err != nil {
		return nil, err
	}
	intParent, err := CreateInternalInstallation(ctx, parentCtx, op.ComponentsRegistry(), parent)
	if err != nil {
		return nil, err
	}
	return intParent, err
}

// GetParentBase returns the parent of an installation base.
// It returns nil if the installation has no parent
func GetParentBase(ctx context.Context, kubeClient client.Client, inst *InstallationBase) (*InstallationBase, error) {
	if IsRootInstallation(inst.Info) {
		return nil, nil
	}
	// get the parent by owner reference
	parentName := GetParentInstallationName(inst.Info)
	parent := &lsv1alpha1.Installation{}
	if err := kubeClient.Get(ctx, client.ObjectKey{Name: parentName, Namespace: inst.Info.Namespace}, parent); err != nil {
		return nil, err
	}
	intParent := CreateInternalInstallationBase(inst.Context, parent)
	return intParent, nil
}

// IsRoot returns if the current component is a root component
func (o *Operation) IsRoot() bool {
	return o.Scope().Parent == nil
}
