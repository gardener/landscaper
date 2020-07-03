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

package subinstallations

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/utils"
)

// TriggerSubInstallations triggers a reconcile for all sub installation of the component.
func (o *Operation) TriggerSubInstallations(ctx context.Context, inst *lsv1alpha1.Installation, operation lsv1alpha1.Operation) error {
	for _, instRef := range inst.Status.InstallationReferences {
		subInst := &lsv1alpha1.Installation{}
		if err := o.Client().Get(ctx, instRef.Reference.NamespacedName(), subInst); err != nil {
			return errors.Wrapf(err, "unable to get sub installation %s", instRef.Reference.NamespacedName().String())
		}

		metav1.SetMetaDataAnnotation(&subInst.ObjectMeta, lsv1alpha1.OperationAnnotation, string(operation))
		if err := o.Client().Update(ctx, subInst); err != nil {
			return errors.Wrapf(err, "unable to update sub installation %s", instRef.Reference.NamespacedName().String())
		}
	}
	return nil
}

// EnsureSubInstallations ensures that all referenced definitions are mapped to a installation.
func (o *Operation) Ensure(ctx context.Context, inst *lsv1alpha1.Installation, def *lsv1alpha1.ComponentDefinition) error {
	cond := lsv1alpha1helper.GetOrInitCondition(inst.Status.Conditions, lsv1alpha1.EnsureSubInstallationsCondition)

	subInstallations, err := o.GetSubInstallations(ctx, inst)
	if err != nil {
		return err
	}

	// need to check if we are allowed to update subinstallation
	// - we are not allowed if any subresource is in deletion
	// - we are not allowed to update if any subinstallation is progressing
	for _, subInstallations := range subInstallations {
		if subInstallations.DeletionTimestamp != nil {
			o.Log().V(7).Info("not eligible for update due to deletion of subinstallation", "name", subInstallations.Name)
			return o.UpdateInstallationStatus(ctx, inst, lsv1alpha1.ComponentPhaseProgressing, cond)
		}

		if subInstallations.Status.Phase == lsv1alpha1.ComponentPhaseProgressing {
			o.Log().V(7).Info("not eligible for update due to running subinstallation", "name", subInstallations.Name)
			return o.UpdateInstallationStatus(ctx, inst, lsv1alpha1.ComponentPhaseProgressing, cond)
		}
	}

	// delete removed subreferences
	err, deleted := o.cleanupOrphanedSubInstallations(ctx, def, inst, subInstallations)
	if err != nil {
		return err
	}
	if deleted {
		return nil
	}

	for _, subDef := range def.DefinitionReferences {
		// skip if the subInstallation already exists
		subInst, ok := subInstallations[subDef.Name]
		if ok {
			if !installationNeedsUpdate(subDef, subInst) {
				continue
			}
		}

		subInst, err := o.createOrUpdateNewInstallation(ctx, inst, def, subDef, subInst)
		if err != nil {
			return errors.Wrapf(err, "unable to create installation for %s", subDef.Name)
		}
	}

	cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue,
		"InstallationsInstalled", "All Installations are successfully installed")
	return o.UpdateInstallationStatus(ctx, inst, inst.Status.Phase, cond)
}

func (o *Operation) GetSubInstallations(ctx context.Context, inst *lsv1alpha1.Installation) (map[string]*lsv1alpha1.Installation, error) {
	var (
		cond             = lsv1alpha1helper.GetOrInitCondition(inst.Status.Conditions, lsv1alpha1.EnsureSubInstallationsCondition)
		subInstallations = map[string]*lsv1alpha1.Installation{}

		// track all found subinstallation to track if some installations were deleted
		updatedSubInstallationStates = make([]lsv1alpha1.NamedObjectReference, 0)
	)

	for _, installationRef := range inst.Status.InstallationReferences {
		subInst := &lsv1alpha1.Installation{}
		if err := o.Client().Get(ctx, installationRef.Reference.NamespacedName(), subInst); err != nil {
			if !apierrors.IsNotFound(err) {
				o.Log().Error(err, "unable to get installation", "object", installationRef.Reference)
				cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
					"InstallationNotFound", fmt.Sprintf("Sub Installation %s not available", installationRef.Reference.Name))
				_ = o.UpdateInstallationStatus(ctx, inst, lsv1alpha1.ComponentPhaseProgressing, cond)
				return nil, errors.Wrapf(err, "unable to get installation %v", installationRef.Reference)
			}
			continue
		}
		subInstallations[installationRef.Name] = subInst
		updatedSubInstallationStates = append(updatedSubInstallationStates, installationRef)
	}

	// update the sub components if installations changed
	if len(updatedSubInstallationStates) != len(inst.Status.InstallationReferences) {
		if err := o.Client().Status().Update(ctx, inst); err != nil {
			return nil, errors.Wrapf(err, "unable to update sub installation status")
		}
	}
	return subInstallations, nil
}

func (o *Operation) cleanupOrphanedSubInstallations(ctx context.Context, def *lsv1alpha1.ComponentDefinition, inst *lsv1alpha1.Installation, subInstallations map[string]*lsv1alpha1.Installation) (error, bool) {
	var (
		cond    = lsv1alpha1helper.GetOrInitCondition(inst.Status.Conditions, lsv1alpha1.EnsureSubInstallationsCondition)
		deleted = false
	)

	for defName, subInst := range subInstallations {
		if _, ok := getDefinitionReference(def, defName); ok {
			continue
		}

		// delete installation
		o.Log().V(5).Info("delete orphaned installation", "name", subInst.Name)
		if err := o.Client().Delete(ctx, subInst); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				"InstallationNotDeleted", fmt.Sprintf("Sub Installation %s cannot be deleted", subInst.Name))
			_ = o.UpdateInstallationStatus(ctx, inst, lsv1alpha1.ComponentPhaseFailed, cond)
			return err, deleted
		}
		deleted = true
	}
	return nil, deleted
}

func (o *Operation) createOrUpdateNewInstallation(ctx context.Context, inst *lsv1alpha1.Installation, def *lsv1alpha1.ComponentDefinition, subDefRef lsv1alpha1.DefinitionReference, subInst *lsv1alpha1.Installation) (*lsv1alpha1.Installation, error) {
	cond := lsv1alpha1helper.GetOrInitCondition(inst.Status.Conditions, lsv1alpha1.EnsureSubInstallationsCondition)

	if subInst == nil {
		subInst = &lsv1alpha1.Installation{}
		subInst.Name = fmt.Sprintf("%s-%s-", def.Name, subDefRef.Name)
		subInst.Namespace = inst.Namespace
	}

	subDef, err := o.Registry().GetDefinitionByRef(subDefRef.Reference)
	if err != nil {
		cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			"ComponentDefinitionNotFound",
			fmt.Sprintf("ComponentDefinition %s for %s cannot be found", subDefRef.Reference, subDefRef.Name))
		_ = o.UpdateInstallationStatus(ctx, inst, lsv1alpha1.ComponentPhaseFailed, cond)
		return nil, errors.Wrapf(err, "unable to get definition %s for %s", subDefRef.Reference, subDefRef.Name)
	}

	_, err = controllerruntime.CreateOrUpdate(ctx, o.Client(), subInst, func() error {
		subInst.Labels = map[string]string{lsv1alpha1.EncompassedByLabel: inst.Name}
		if err := controllerutil.SetOwnerReference(inst, subInst, o.Scheme()); err != nil {
			return errors.Wrapf(err, "unable to set owner reference")
		}
		subInst.Spec = lsv1alpha1.InstallationSpec{
			DefinitionRef: subDefRef.Reference,
			Imports:       subDefRef.Imports,
			Exports:       subDefRef.Exports,
		}

		AddDefaultMappings(subInst, subDef)
		return nil
	})
	if err != nil {
		cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			"InstallationCreatingFailed",
			fmt.Sprintf("Sub Installation %s cannot be created", subDefRef.Name))
		_ = o.UpdateInstallationStatus(ctx, inst, lsv1alpha1.ComponentPhaseFailed, cond)
		return nil, errors.Wrapf(err, "unable to create installation for %s", subDefRef.Name)
	}

	// add newly created installation to state
	inst.Status.InstallationReferences = append(inst.Status.InstallationReferences, lsv1alpha1helper.NewInstallationReferenceState(subDefRef.Name, subInst))
	if err := o.Client().Status().Update(ctx, inst); err != nil {
		return nil, errors.Wrapf(err, "unable to add new installation for %s to state", subDefRef.Name)
	}

	return subInst, nil
}

// GetExportedValues returns the merged export of all subinstallations
func (o *Operation) GetExportedValues(ctx context.Context, inst *installations.Installation) (map[string]interface{}, error) {
	values := make(map[string]interface{}, 0)

	subInstallations, err := o.GetSubInstallations(ctx, inst.Info)
	if err != nil {
		return nil, err
	}

	for _, subInst := range subInstallations {
		if subInst.Status.ExportReference == nil {
			continue
		}
		do, err := o.Operation.GetDataObjectFromSecret(ctx, subInst.Status.ExportReference.NamespacedName())
		if err != nil {
			return nil, err
		}

		values = utils.MergeMaps(values, do.Data)
	}

	return values, nil
}
