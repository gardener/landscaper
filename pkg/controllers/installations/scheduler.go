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
func (a *actuator) importsAreSatisfied(ctx context.Context, landscapeConfig map[string]interface{}, inst *lsv1alpha1.ComponentInstallation) error {
	return nil
}

// determineInstallationContext determines the visible context of the current component.
// The visible context consists of the installation's parent and siblings.
func (a *actuator) determineInstallationContext(ctx context.Context, inst *lsv1alpha1.ComponentInstallation) (*lsv1alpha1.ComponentInstallation, []*lsv1alpha1.ComponentInstallation, error) {
	if IsRootInstallation(inst) {
		// get all root object as siblings
		ownInstSelector := client.MatchingFieldsSelector{Selector: fields.OneTermNotEqualSelector("metadata.name", inst.Name)}
		installations, err := a.GetRootInstallations(ctx, client.InNamespace(inst.Namespace), ownInstSelector)
		if err != nil {
			return nil, nil, err
		}
		return nil, installations, nil
	}

	// get the parent by owner reference
	parentName := GetParentInstallationName(inst)
	parent := &lsv1alpha1.ComponentInstallation{}
	if err := a.c.Get(ctx, client.ObjectKey{Name: parentName, Namespace: inst.Namespace}, parent); err != nil {
		return nil, nil, err
	}

	// siblings are all encompassed installation of the parent installation
	subInstallations := make([]*lsv1alpha1.ComponentInstallation, 0)
	for _, installationRef := range parent.Status.InstallationReferences {
		if installationRef.Reference.Name == inst.Name {
			continue
		}
		subInst := &lsv1alpha1.ComponentInstallation{}
		if err := a.c.Get(ctx, installationRef.Reference.NamespacedName(), subInst); err != nil {
			return nil, nil, err
		}
		subInstallations = append(subInstallations, subInst)
	}

	return parent, subInstallations, nil
}
