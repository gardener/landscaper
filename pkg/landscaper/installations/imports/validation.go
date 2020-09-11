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

	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"
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

// Validate traverses through all components and validates if all imports are
// satisfied with the correct version
func (v *Validator) Validate(ctx context.Context, inst *installations.Installation) error {
	cond := lsv1alpha1helper.GetOrInitCondition(inst.Info.Status.Conditions, lsv1alpha1.ValidateImportsCondition)
	fldPath := field.NewPath(inst.Info.Name)

	// check if parent has sibling installation dependencies that are not finished yet
	completed, err := installations.CheckCompletedSiblingDependentsOfParent(ctx, v, v.parent)
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

	for _, importMapping := range inst.GetImportMappings() {
		impPath := fldPath.Child(importMapping.Name)
		err = v.checkImportMappingIsSatisfied(ctx, impPath, inst, importMapping)
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

func (v *Validator) checkImportMappingIsSatisfied(ctx context.Context, fldPath *field.Path, inst *installations.Installation, mapping installations.ImportMapping) error {
	err := v.checkStaticDataForMapping(ctx, fldPath, inst, mapping)
	if !installations.IsImportNotFoundError(err) {
		return err
	}
	if err == nil {
		return nil
	}

	// get deploy item from current context
	// todo: find better name
	raw := &lsv1alpha1.DataObject{}
	doName := lsv1alpha1helper.GenerateDataObjectName(v.Context().Name, mapping.From)
	if err := v.Client().Get(ctx, kutil.ObjectKey(doName, inst.Info.Namespace), raw); err != nil {
		return err
	}
	do, err := dataobjects.NewFromDataObject(raw)
	if err != nil {
		return err
	}

	if err := v.JSONSchemaValidator().ValidateGoStruct(mapping.Schema, do.Data); err != nil {
		return installations.NewImportNotSatisfiedErrorf(err, "%s: imported datatype does not have the expected schema: %s", fldPath.String(), err.Error())
	}

	owner := kutil.GetOwner(do.Raw.ObjectMeta)
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
		return v.checkStateForParentImport(fldPath, mapping)
	}

	// otherwise validate as sibling export
	return v.checkStateForSiblingExport(ctx, fldPath, ref, mapping)
}

func (v *Validator) checkStaticDataForMapping(ctx context.Context, fldPath *field.Path, inst *installations.Installation, mapping installations.ImportMapping) error {
	if inst.Info.Spec.StaticData == nil {
		return installations.NewImportNotFoundErrorf(nil, "%s: static data not defined", fldPath.String())
	}

	data, err := v.GetStaticData(ctx)
	if err != nil {
		return err
	}

	var val interface{}
	if err := jsonpath.GetValue(mapping.From, data, &val); err != nil {
		return installations.NewImportNotFoundErrorf(err, "%s: import in landscape config not found", fldPath.String())
	}

	if err := v.JSONSchemaValidator().ValidateGoStruct(mapping.Schema, val); err != nil {
		return installations.NewImportNotSatisfiedErrorf(err, "%s: imported datatype does not have the expected schema: %s", fldPath.String(), err.Error())
	}
	return nil
}

func (v *Validator) checkStateForParentImport(fldPath *field.Path, mapping installations.ImportMapping) error {
	// check if the parent also imports my import
	_, err := v.parent.GetImportDefinition(mapping.From)
	if err != nil {
		return installations.NewImportNotFoundErrorf(err, "%s: import in parent not found", fldPath.String())
	}
	// parent has to be progressing
	if v.parent.Info.Status.Phase != lsv1alpha1.ComponentPhaseProgressing {
		return installations.NewImportNotSatisfiedErrorf(nil, "%s: Parent has to be progressing to get imports", fldPath.String())
	}
	return nil
}

func (v *Validator) checkStateForSiblingExport(ctx context.Context, fldPath *field.Path, siblingRef lsv1alpha1.ObjectReference, mapping installations.ImportMapping) error {
	sibling := v.getSiblingForObjectReference(siblingRef)
	if sibling == nil {
		return fmt.Errorf("%s: installation %s is not a sibling", fldPath.String(), siblingRef.NamespacedName().String())
	}

	// search in the sibling for the export mapping where importmap.from == exportmap.to
	_, err := sibling.GetExportMappingTo(mapping.From)
	if err != nil {
		return installations.NewImportNotFoundErrorf(err, "%s: export in sibling not found", fldPath.String())
	}

	if sibling.Info.Status.Phase != lsv1alpha1.ComponentPhaseSucceeded {
		return installations.NewNotCompletedDependentsErrorf(nil, "%s: Sibling Installation has to be completed to get exports", fldPath.String())
	}

	// todo: check generation of other components in the dependency tree
	// we expect that no dependent siblings are running
	isCompleted, err := installations.CheckCompletedSiblingDependents(ctx, v, sibling)
	if err != nil {
		return fmt.Errorf("%s: Unable to check if sibling Installation dependencies are not completed yet", fldPath.String())
	}
	if !isCompleted {
		return installations.NewNotCompletedDependentsErrorf(nil, "%s: A sibling Installation dependency is not completed yet", fldPath.String())
	}

	return nil
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
