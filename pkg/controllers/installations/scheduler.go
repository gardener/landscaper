// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package installations

import (
	"context"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/pkg/landscaper/component"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// triggerSubInstallations triggers a reconcile for all sub installation of the component.
func (a *actuator) triggerSubInstallations(ctx context.Context, inst *lsv1alpha1.ComponentInstallation) error {
	for _, instRef := range inst.Status.InstallationReferences {
		subInst := &lsv1alpha1.ComponentInstallation{}
		if err := a.c.Get(ctx, instRef.Reference.NamespacedName(), subInst); err != nil {
			return errors.Wrapf(err, "unable to get sub installation %s", instRef.Reference.NamespacedName().String())
		}

		metav1.SetMetaDataAnnotation(&subInst.ObjectMeta, lsv1alpha1.OperationAnnotation, string(lsv1alpha1.ReconcileOperation))
		if err := a.c.Update(ctx, subInst); err != nil {
			return errors.Wrapf(err, "unable to update sub installation %s", instRef.Reference.NamespacedName().String())
		}
	}
	return nil
}

// importsAreSatisfied traverses through all components and validates if all imports are
// satisfied with the correct version
func (a *actuator) importsAreSatisfied(ctx context.Context, landscapeConfig *lsv1alpha1.LandscapeConfiguration, def *lsv1alpha1.ComponentDefinition, inst *lsv1alpha1.ComponentInstallation, lsCtx *Context) (bool, error) {
	var (
		parent = lsCtx.Parent
		//siblings = lsCtx.Siblings
	)

	inInst, err := component.New(inst, def)
	if err != nil {
		return false, err
	}

	for _, importMapping := range inst.Spec.Imports {
		importDef, err := inInst.GetImportDefinition(importMapping.To)
		// check landscape config if I'm a root installation

		// check if the parent also imports my import
		parentImport, err := parent.GetImportDefinition(importMapping.From)
		if err != nil {
			return false, err
		}

		// parent has to be progressing
		if parent.Info.Status.Phase != lsv1alpha1.ComponentPhaseProgressing {
			return false, errors.New("Parent has to be progressing to get imports")
		}

		if parentImport.Type != importDef.Type {
			return false, errors.New("abc")
		}

		// check if a siblings exports the given value

	}

	return false, nil
}

// determineInstallationContext determines the visible context of a installation.
// The visible context consists of the installation's parent and siblings.
// The context is later used to validate and get imported data.
func (a *actuator) determineInstallationContext(ctx context.Context, inst *lsv1alpha1.ComponentInstallation) (*Context, error) {
	if IsRootInstallation(inst) {
		// get all root object as siblings
		ownInstSelector := client.MatchingFieldsSelector{Selector: fields.OneTermNotEqualSelector("metadata.name", inst.Name)}
		installations, err := a.GetRootInstallations(ctx, client.InNamespace(inst.Namespace), ownInstSelector)
		if err != nil {
			return nil, err
		}
		intInstallations, err := CreateInternalInstallations(a.registry, installations...)
		if err != nil {
			return nil, err
		}
		return &Context{Siblings: intInstallations}, nil
	}

	// get the parent by owner reference
	parentName := GetParentInstallationName(inst)
	parent := &lsv1alpha1.ComponentInstallation{}
	if err := a.c.Get(ctx, client.ObjectKey{Name: parentName, Namespace: inst.Namespace}, parent); err != nil {
		return nil, err
	}

	// siblings are all encompassed installation of the parent installation
	subInstallations := make([]*lsv1alpha1.ComponentInstallation, 0)
	for _, installationRef := range parent.Status.InstallationReferences {
		if installationRef.Reference.Name == inst.Name {
			continue
		}
		subInst := &lsv1alpha1.ComponentInstallation{}
		if err := a.c.Get(ctx, installationRef.Reference.NamespacedName(), subInst); err != nil {
			return nil, err
		}
		subInstallations = append(subInstallations, subInst)
	}

	intParent, err := CreateInternalInstallation(a.registry, parent)
	if err != nil {
		return nil, err
	}

	intSubInstallations, err := CreateInternalInstallations(a.registry, subInstallations...)
	if err != nil {
		return nil, err
	}

	return &Context{Parent: intParent, Siblings: intSubInstallations}, nil
}
