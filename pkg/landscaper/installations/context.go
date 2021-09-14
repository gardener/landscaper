// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
)

// Context contains the visible installations of a specific installation.
// This context is later used to validate and get import data
type Context struct {
	// Name is the name of the current installation's context.
	// By default, it is the source name of the parent.
	Name string
	// Parent is the installation is encompassed in.
	// Parents are handled separately as installation have access to the same imports as their parent.
	Parent *Installation

	// Siblings are installations with the same parent.
	// The installation has access to the exports of these components
	Siblings []*InstallationBase
}

// SetInstallationContext determines the current context and updates the operation context.
func (o *Operation) SetInstallationContext(ctx context.Context) error {
	newCtx, err := o.DetermineInstallationContext(ctx)
	if err != nil {
		return err
	}
	o.context = *newCtx
	return nil
}

// DetermineInstallationContext determines the visible context of an installation.
// The visible context consists of the installation's parent and siblings.
// The context is later used to validate and get imported data.
func (o *Operation) DetermineInstallationContext(ctx context.Context) (*Context, error) {
	parentInst, siblingInstallations, err := GetParentAndSiblings(ctx, o.Client(), o.Inst.Info)
	if err != nil {
		return nil, err
	}

	// set optional default repository context
	for _, inst := range siblingInstallations {
		if inst.Spec.ComponentDescriptor != nil &&
			inst.Spec.ComponentDescriptor.Reference != nil &&
			inst.Spec.ComponentDescriptor.Reference.RepositoryContext == nil {
			inst.Spec.ComponentDescriptor.Reference.RepositoryContext = o.DefaultRepoContext
		}
	}

	// get the parent by owner reference
	parent, err := CreateInternalInstallation(ctx, o.ComponentsRegistry(), parentInst)
	if err != nil {
		return nil, err
	}

	ctxName := ""
	if parentInst != nil {
		ctxName = lsv1alpha1helper.DataObjectSourceFromInstallation(parentInst)
	}

	return &Context{
		Name:   ctxName,
		Parent: parent,
		// siblings are all encompassed installation of the parent installation
		Siblings: CreateInternalInstallationBases(siblingInstallations...),
	}, nil
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
	if err := kubeClient.Get(ctx, client.ObjectKey{Name: parentName, Namespace: inst.Namespace}, parent); err != nil {
		return nil, err
	}
	return parent, nil
}

// IsRoot returns if the current component is a root component
func (o *Operation) IsRoot() bool {
	return o.Context().Parent == nil
}
