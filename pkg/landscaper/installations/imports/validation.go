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

package imports

import (
	"context"
	"fmt"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// CheckSiblingDependentsOfParentsReason defines the reason for the parent dependent validation
const CheckSiblingDependentsOfParentsReason = "CheckSiblingDependentsOfParentsReason"

// CheckImportsReason defines the reason for the import validation
const CheckImportsReason = "CheckImportsReason"

// NewValidator creates new import validator.
// It validates if all imports of a component are satisfied given a context.
func NewValidator(op *installations.Operation) *Validator {
	return &Validator{
		Operation: op,
		parent:    op.Context().Parent,
		siblings:  op.Context().Siblings,
	}
}

// OutdatedImports validates whether a imported data object or target is outdated.
func (v *Validator) OutdatedImports(ctx context.Context, inst *installations.Installation) (bool, error) {
	fldPath := field.NewPath(fmt.Sprintf("(Inst %s)", inst.Info.Name))

	for _, dataImport := range inst.Info.Spec.Imports.Data {
		impPath := fldPath.Child(dataImport.Name)
		outdated, err := v.checkDataImportIsOutdated(ctx, impPath, inst, dataImport)
		if err != nil {
			return false, err
		}
		if outdated {
			return true, nil
		}
	}

	for _, targetImport := range inst.Info.Spec.Imports.Targets {
		impPath := fldPath.Child(targetImport.Name)
		outdated, err := v.checkTargetImportIsOutdated(ctx, impPath, inst, targetImport)
		if err != nil {
			return false, err
		}
		if outdated {
			return true, nil
		}
	}

	return false, nil
}

// Validate traverses through all components and validates if all imports are
// satisfied with the correct version
func (v *Validator) Validate(ctx context.Context, inst *installations.Installation) error {
	cond := lsv1alpha1helper.GetOrInitCondition(inst.Info.Status.Conditions, lsv1alpha1.ValidateImportsCondition)
	fldPath := field.NewPath(fmt.Sprintf("(Inst %s)", inst.Info.Name))

	// check if parent has sibling installation dependencies that are not finished yet
	completed, err := CheckCompletedSiblingDependentsOfParent(ctx, v.Operation, v.parent)
	if err != nil {
		inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			CheckSiblingDependentsOfParentsReason,
			fmt.Sprintf("Check for progressing dependents of the parent failed: %s", err.Error())))
		return err
	}
	if !completed {
		inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			CheckSiblingDependentsOfParentsReason,
			"Waiting until all progressing dependents of the parent are finished"))
		return installations.NewNotCompletedDependentsError("A parent or parent's parent sibling Installation dependency is not completed yet", nil)
	}

	for _, dataImport := range inst.Info.Spec.Imports.Data {
		impPath := fldPath.Child(dataImport.Name)
		err = v.checkDataImportIsSatisfied(ctx, impPath, inst, dataImport)
		if err != nil {
			inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				CheckImportsReason,
				fmt.Sprintf("Waiting until all imports are satisfied: %s", err.Error())))
			return err
		}
	}

	for _, targetImport := range inst.Info.Spec.Imports.Targets {
		impPath := fldPath.Child(targetImport.Name)
		err = v.checkTargetImportIsSatisfied(ctx, impPath, inst, targetImport)
		if err != nil {
			inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				CheckImportsReason,
				fmt.Sprintf("Waiting until all imports are satisfied: %s", err.Error())))
			return err
		}
	}

	inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue,
		CheckImportsReason,
		"All imports are satisfied"))
	return nil
}

func (v *Validator) checkDataImportIsOutdated(ctx context.Context, fldPath *field.Path, inst *installations.Installation, dataImport lsv1alpha1.DataImport) (bool, error) {
	// get deploy item from current context
	do, owner, err := v.getDataImport(ctx, inst, dataImport.DataRef)
	if err != nil {
		return false, fmt.Errorf("%s: unable to get data object for '%s': %w", fldPath.String(), dataImport.Name, err)
	}
	importStatus, err := inst.ImportStatus().GetData(dataImport.Name)
	if err != nil {
		return true, nil
	}

	// we cannot validate if the source is not an installation
	if owner == nil || owner.Kind != "Installation" {
		if strconv.Itoa(int(do.Raw.Generation)) != importStatus.ConfigGeneration {
			return true, nil
		}

		return false, nil
	}

	ref := lsv1alpha1.ObjectReference{Name: owner.Name, Namespace: inst.Info.Namespace}
	src := &lsv1alpha1.Installation{}
	if err := v.Client().Get(ctx, ref.NamespacedName(), src); err != nil {
		return false, fmt.Errorf("%s: unable to get source installation %s for '%s': %w", fldPath.String(), ref.NamespacedName().String(), dataImport.Name, err)
	}
	if src.Status.ConfigGeneration != importStatus.ConfigGeneration {
		return true, nil
	}
	return false, nil
}

func (v *Validator) checkTargetImportIsOutdated(ctx context.Context, fldPath *field.Path, inst *installations.Installation, targetImport lsv1alpha1.TargetImportExport) (bool, error) {
	// get deploy item from current context
	target, owner, err := v.getTargetImport(ctx, inst, targetImport.Target)
	if err != nil {
		return false, fmt.Errorf("%s: unable to get data object for '%s': %w", fldPath.String(), targetImport.Name, err)
	}
	importStatus, err := inst.ImportStatus().GetData(targetImport.Name)
	if err != nil {
		return true, nil
	}

	// we cannot validate if the source is not an installation
	if owner == nil || owner.Kind != "Installation" {
		if strconv.Itoa(int(target.Generation)) != importStatus.ConfigGeneration {
			return true, nil
		}

		return false, nil
	}

	ref := lsv1alpha1.ObjectReference{Name: owner.Name, Namespace: inst.Info.Namespace}
	src := &lsv1alpha1.Installation{}
	if err := v.Client().Get(ctx, ref.NamespacedName(), src); err != nil {
		return false, fmt.Errorf("%s: unable to get source installation %s for '%s': %w", fldPath.String(), ref.NamespacedName().String(), targetImport.Name, err)
	}
	if src.Status.ConfigGeneration != importStatus.ConfigGeneration {
		return true, nil
	}
	return false, nil
}

func (v *Validator) checkDataImportIsSatisfied(ctx context.Context, fldPath *field.Path, inst *installations.Installation, dataImport lsv1alpha1.DataImport) error {
	// get deploy item from current context
	_, owner, err := v.getDataImport(ctx, inst, dataImport.DataRef)
	if err != nil {
		return fmt.Errorf("%s: unable to get data object for '%s': %w", fldPath.String(), dataImport.Name, err)
	}
	if owner == nil {
		return nil
	}

	// we cannot validate if the source is not an installation
	if owner.Kind != "Installation" {
		return nil
	}
	ref := lsv1alpha1.ObjectReference{Name: owner.Name, Namespace: inst.Info.Namespace}

	// check if the data object comes from the parent
	if lsv1alpha1helper.ReferenceIsObject(ref, v.parent.Info) {
		return v.checkStateForParentImport(fldPath, dataImport.DataRef)
	}

	// otherwise validate as sibling export
	return v.checkStateForSiblingDataExport(ctx, fldPath, ref, dataImport.DataRef)
}

func (v *Validator) checkTargetImportIsSatisfied(ctx context.Context, fldPath *field.Path, inst *installations.Installation, targetImport lsv1alpha1.TargetImportExport) error {
	// get deploy item from current context
	_, owner, err := v.getTargetImport(ctx, inst, targetImport.Target)
	if err != nil {
		return fmt.Errorf("%s: unable to get target for '%s': %w", fldPath.String(), targetImport.Name, err)
	}
	if owner == nil {
		return nil
	}

	// we cannot validate if the source is not an installation
	if owner.Kind != "Installation" {
		return nil
	}
	ref := lsv1alpha1.ObjectReference{Name: owner.Name, Namespace: inst.Info.Namespace}

	// check if the data object comes from the parent
	if lsv1alpha1helper.ReferenceIsObject(ref, v.parent.Info) {
		return v.checkStateForParentImport(fldPath, targetImport.Target)
	}

	// otherwise validate as sibling export
	return v.checkStateForSiblingDataExport(ctx, fldPath, ref, targetImport.Target)
}

func (v *Validator) checkStateForParentImport(fldPath *field.Path, importName string) error {
	// check if the parent also imports my import
	_, err := v.parent.GetImportDefinition(importName)
	if err != nil {
		return installations.NewImportNotFoundErrorf(err, "%s: import in parent not found", fldPath.String())
	}
	// parent has to be progressing
	if v.parent.Info.Status.Phase != lsv1alpha1.ComponentPhaseProgressing {
		return installations.NewImportNotSatisfiedErrorf(nil, "%s: Parent has to be progressing to get imports", fldPath.String())
	}
	return nil
}

func (v *Validator) checkStateForSiblingDataExport(ctx context.Context, fldPath *field.Path, siblingRef lsv1alpha1.ObjectReference, importName string) error {
	sibling := v.getSiblingForObjectReference(siblingRef)
	if sibling == nil {
		return fmt.Errorf("%s: installation %s is not a sibling", fldPath.String(), siblingRef.NamespacedName().String())
	}

	// search in the sibling for the export mapping where importmap.from == exportmap.to
	if !sibling.IsExportingData(importName) {
		return installations.NewImportNotFoundErrorf(nil, "%s: export in sibling not found", fldPath.String())
	}

	if sibling.Info.Status.Phase != lsv1alpha1.ComponentPhaseSucceeded {
		return installations.NewNotCompletedDependentsErrorf(nil, "%s: Sibling Installation has to be completed to get exports", fldPath.String())
	}

	// todo: check generation of other components in the dependency tree
	// we expect that no dependent siblings are running
	isCompleted, err := CheckCompletedSiblingDependents(ctx, v.Operation, sibling)
	if err != nil {
		return fmt.Errorf("%s: Unable to check if sibling Installation dependencies are not completed yet", fldPath.String())
	}
	if !isCompleted {
		return installations.NewNotCompletedDependentsErrorf(nil, "%s: A sibling Installation dependency is not completed yet", fldPath.String())
	}

	return nil
}

func (v *Validator) getDataImport(ctx context.Context, inst *installations.Installation, dataRef string) (*dataobjects.DataObject, *metav1.OwnerReference, error) {
	// get deploy item from current context
	raw := &lsv1alpha1.DataObject{}
	doName := lsv1alpha1helper.GenerateDataObjectName(v.Context().Name, dataRef)
	if err := v.Client().Get(ctx, kutil.ObjectKey(doName, inst.Info.Namespace), raw); err != nil {
		return nil, nil, err
	}
	do, err := dataobjects.NewFromDataObject(raw)
	if err != nil {
		return nil, nil, err
	}

	owner := kutil.GetOwner(do.Raw.ObjectMeta)
	return do, owner, nil
}

func (v *Validator) getTargetImport(ctx context.Context, inst *installations.Installation, target string) (*lsv1alpha1.Target, *metav1.OwnerReference, error) {
	// get deploy item from current context
	raw := &lsv1alpha1.Target{}
	targetName := lsv1alpha1helper.GenerateDataObjectName(v.Context().Name, target)
	if err := v.Client().Get(ctx, kutil.ObjectKey(targetName, inst.Info.Namespace), raw); err != nil {
		return nil, nil, err
	}

	owner := kutil.GetOwner(raw.ObjectMeta)
	return raw, owner, nil
}

func (v *Validator) getSiblingForObjectReference(ref lsv1alpha1.ObjectReference) *installations.Installation {
	for _, sibling := range v.siblings {
		if lsv1alpha1helper.ReferenceIsObject(ref, sibling.Info) {
			return sibling
		}
	}
	return nil
}

// IsRoot returns true if the current component is a root component
func (v *Validator) IsRoot() bool {
	return v.parent == nil
}
