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
	"fmt"

	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	landscaperv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	landscaperv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
)

// EnsureSubInstallations ensures that all referenced definitions are mapped to a installation.
func (a *actuator) EnsureSubInstallations(ctx context.Context, inst *landscaperv1alpha1.ComponentInstallation, def *landscaperv1alpha1.ComponentDefinition) error {
	cond := landscaperv1alpha1helper.GetOrInitCondition(inst.Status.Conditions, landscaperv1alpha1.EnsureSubInstallationsCondition)

	subInstallations, err := a.getSubInstallations(ctx, inst)
	if err != nil {
		return err
	}

	for _, subDef := range def.DefinitionReferences {
		// skip if the subInstallation already exists
		subInst, ok := subInstallations[subDef.Name]
		if ok {
			if !installationNeedsUpdate(subDef, subInst) {
				continue
			}
		}

		subInst, err := a.createNewInstallation(ctx, inst, def, subDef, subInst)
		if err != nil {
			return errors.Wrapf(err, "unable to create installation for %s", subDef.Name)
		}

		// add newly created installation to state
		inst.Status.InstallationReferences = append(inst.Status.InstallationReferences, landscaperv1alpha1helper.NewInstallationReferenceState(subDef.Name, subInst))
		if err := a.c.Status().Update(ctx, inst); err != nil {
			return errors.Wrapf(err, "unable to add new installation for %s to state", subDef.Name)
		}
	}

	cond = landscaperv1alpha1helper.UpdatedCondition(cond, landscaperv1alpha1.ConditionTrue,
		"InstallationsInstalled", "All Installations are successfully installed")
	return a.updateInstallationStatus(ctx, inst, cond)
}

func (a *actuator) getSubInstallations(ctx context.Context, inst *landscaperv1alpha1.ComponentInstallation) (map[string]*landscaperv1alpha1.ComponentInstallation, error) {
	var (
		cond             = landscaperv1alpha1helper.GetOrInitCondition(inst.Status.Conditions, landscaperv1alpha1.EnsureSubInstallationsCondition)
		subInstallations = map[string]*landscaperv1alpha1.ComponentInstallation{}

		// track all found subinstallation to track if some installations were deleted
		updatedSubInstallationStates = make([]landscaperv1alpha1.NamedObjectReference, 0)
	)

	for _, installationRef := range inst.Status.InstallationReferences {
		subInst := &landscaperv1alpha1.ComponentInstallation{}
		if err := a.c.Get(ctx, installationRef.Reference.NamespacedName(), subInst); err != nil {
			if !apierrors.IsNotFound(err) {
				a.log.Error(err, "unable to get installation", "object", installationRef.Reference)
				cond = landscaperv1alpha1helper.UpdatedCondition(cond, landscaperv1alpha1.ConditionFalse,
					"InstallationNotFound", fmt.Sprintf("Sub Installation %s not available", installationRef.Reference.Name))
				_ = a.updateInstallationStatus(ctx, inst, cond)
				return nil, errors.Wrapf(err, "unable to get installation %v", installationRef.Reference)
			}
			continue
		}
		subInstallations[installationRef.Name] = subInst
		updatedSubInstallationStates = append(updatedSubInstallationStates, installationRef)
	}

	// update the sub components if installations changed
	if len(updatedSubInstallationStates) != len(inst.Status.InstallationReferences) {
		if err := a.c.Status().Update(ctx, inst); err != nil {
			return nil, errors.Wrapf(err, "unable to update sub installation status")
		}
	}
	return subInstallations, nil
}

func (a *actuator) createNewInstallation(ctx context.Context, inst *landscaperv1alpha1.ComponentInstallation, def *landscaperv1alpha1.ComponentDefinition, subDefRef landscaperv1alpha1.DefinitionReference, subInst *landscaperv1alpha1.ComponentInstallation) (*landscaperv1alpha1.ComponentInstallation, error) {
	cond := landscaperv1alpha1helper.GetOrInitCondition(inst.Status.Conditions, landscaperv1alpha1.EnsureSubInstallationsCondition)

	if subInst == nil {
		subInst = &landscaperv1alpha1.ComponentInstallation{}
		subInst.Name = fmt.Sprintf("%s-%s-", def.Name, subDefRef.Name)
		subInst.Namespace = inst.Namespace
		subInst.Labels = map[string]string{landscaperv1alpha1.EncompassedByLabel: inst.Name}
	}

	_, err := controllerruntime.CreateOrUpdate(ctx, a.c, subInst, func() error {
		subInst.Spec = landscaperv1alpha1.ComponentInstallationSpec{
			DefinitionRef: subDefRef.Reference,
			Imports:       subDefRef.Imports,
			Exports:       subDefRef.Exports,
		}
		return nil
	})
	if err != nil {
		cond = landscaperv1alpha1helper.UpdatedCondition(cond, landscaperv1alpha1.ConditionFalse,
			"InstallationCreatingFailed",
			fmt.Sprintf("Sub Installation %s cannot be created", subDefRef.Name))
		_ = a.updateInstallationStatus(ctx, inst, cond)
		return nil, errors.Wrapf(err, "unable to create installation for %s", subDefRef.Name)
	}

	// add newly created installation to state
	inst.Status.InstallationReferences = append(inst.Status.InstallationReferences, landscaperv1alpha1helper.NewInstallationReferenceState(subDefRef.Name, subInst))
	if err := a.c.Status().Update(ctx, inst); err != nil {
		return nil, errors.Wrapf(err, "unable to add new installation for %s to state", subDefRef.Name)
	}

	return subInst, nil
}

// installationNeedsUpdate check if a definition reference has been updated
func installationNeedsUpdate(def landscaperv1alpha1.DefinitionReference, inst *landscaperv1alpha1.ComponentInstallation) bool {
	if def.Reference != inst.Spec.DefinitionRef {
		return true
	}

	for _, mapping := range def.Imports {
		if !hasMappingOfImports(mapping, inst.Spec.Imports) {
			return true
		}
	}

	for _, mapping := range def.Exports {
		if !hasMappingOfExports(mapping, inst.Spec.Exports) {
			return true
		}
	}

	if len(inst.Spec.Imports) != len(def.Imports) {
		return true
	}

	if len(inst.Spec.Exports) != len(def.Exports) {
		return true
	}

	return false
}

func hasMappingOfImports(search landscaperv1alpha1.DefinitionImportMapping, mappings []landscaperv1alpha1.DefinitionImportMapping) bool {
	for _, mapping := range mappings {
		if mapping.To == search.To && mapping.From == search.From {
			return true
		}
	}
	return false
}

func hasMappingOfExports(search landscaperv1alpha1.DefinitionExportMapping, mappings []landscaperv1alpha1.DefinitionExportMapping) bool {
	for _, mapping := range mappings {
		if mapping.To == search.To && mapping.From == search.From {
			return true
		}
	}
	return false
}
