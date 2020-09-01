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

	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"
	"github.com/gardener/landscaper/pkg/landscaper/datatype"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
)

// NewValidator creates new import validator.
// It validates if all imports of a component are satisfied given a context.
func NewValidator(op *installations.Operation, parent *installations.Installation, siblings ...*installations.Installation) *Validator {
	return &Validator{
		Operation: op,

		parent:   parent,
		siblings: siblings,
	}
}

// Validate traverses through all components and validates if all imports are
// satisfied with the correct version
func (v *Validator) Validate(ctx context.Context, inst *installations.Installation) error {
	fldPath := field.NewPath(inst.Info.Name)

	// check if parent has sibling installation dependencies that are not finished yet
	completed, err := installations.CheckCompletedSiblingDependentsOfParent(ctx, v, v.parent)
	if err != nil {
		return err
	}
	if !completed {
		return installations.NewNotCompletedDependentsError("A parent or parent's parent sibling Installation dependency is not completed yet", nil)
	}

	mappings, err := inst.GetImportMappings()
	if err != nil {
		return err
	}
	for _, importMapping := range mappings {
		impPath := fldPath.Child(importMapping.Key)
		err = v.checkImportMappingIsSatisfied(ctx, impPath, inst, importMapping)
		if err != nil {
			return err
		}
	}

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

	if !v.IsRoot() {
		// check if the parent also imports my import
		err = v.checkIfParentHasImportForMapping(fldPath, inst, mapping)
		if !installations.IsImportNotFoundError(err) {
			return err
		}
		if err == nil {
			return nil
		}
	}

	// check if a sibling exports the given value
	err = v.checkIfSiblingsHaveImportForMapping(ctx, fldPath, inst, mapping)
	if !installations.IsImportNotFoundError(err) {
		return err
	}
	if err == nil {
		return nil
	}

	return installations.NewImportNotFoundError("", field.NotFound(fldPath, "no import found"))
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

	// validate types
	dt, ok := v.GetDataType(mapping.Type)
	if !ok {
		return installations.NewImportNotSatisfiedErrorf(nil, "%s: datatype %s cannot be found", fldPath.String(), mapping.Type)
	}
	if err := datatype.Validate(*dt, val); err != nil {
		return installations.NewImportNotSatisfiedErrorf(err, "%s: imported datatype does not have the expected type %s", fldPath.String(), mapping.Type)
	}

	return nil
}

func (v *Validator) checkIfParentHasImportForMapping(fldPath *field.Path, inst *installations.Installation, mapping installations.ImportMapping) error {
	// check if the parent also imports my import
	parentImport, err := v.parent.GetImportDefinition(mapping.From)
	if err != nil {
		return installations.NewImportNotFoundErrorf(err, "%s: ImportDefinition not found", fldPath.String())
	}

	// parent has to be progressing
	if v.parent.Info.Status.Phase != v1alpha1.ComponentPhaseProgressing {
		return installations.NewImportNotSatisfiedErrorf(nil, "%s: Parent has to be progressing to get imports", fldPath.String())
	}

	if parentImport.Type != mapping.Type {
		return installations.NewImportNotSatisfiedErrorf(nil, "%s: import type of parent is %s but expected %s", fldPath.String(), parentImport.Type, mapping.Type)
	}

	return nil
}

func (v *Validator) checkIfSiblingsHaveImportForMapping(ctx context.Context, fldPath *field.Path, inst *installations.Installation, mapping installations.ImportMapping) error {

	for _, sibling := range v.siblings {
		sPath := fldPath.Child(sibling.Info.Name)
		err := v.checkIfSiblingHasImportForMapping(ctx, sPath, inst, mapping, sibling)
		if !installations.IsImportNotFoundError(err) {
			return err
		}
		if err == nil {
			return nil
		}
	}

	return installations.NewImportNotFoundErrorf(nil, "%s: no sibling installation found to satisfy the mapping", fldPath.String())
}

func (v *Validator) checkIfSiblingHasImportForMapping(ctx context.Context, fldPath *field.Path, inst *installations.Installation, mapping installations.ImportMapping, sibling *installations.Installation) error {
	// search in the sibling for the export mapping where importmap.from == exportmap.to
	exportMapping, err := sibling.GetExportMappingTo(mapping.From)
	if err != nil {
		return installations.NewImportNotFoundErrorf(err, "%s: ExportMapping not found in sibling", fldPath.String())
	}

	// check if the sibling also imports my import
	siblingExport, err := sibling.GetExportDefinition(exportMapping.To)
	if err != nil {
		return installations.NewImportNotFoundErrorf(err, "%s: ImportDefinition not found", fldPath.String())
	}

	// sibling exports the key we import

	if sibling.Info.Status.Phase != v1alpha1.ComponentPhaseSucceeded {
		return installations.NewImportNotSatisfiedErrorf(nil, "%s: Sibling Installation has to be completed to get exports", fldPath.String())
	}

	if siblingExport.Type != mapping.Type {
		return installations.NewImportNotSatisfiedErrorf(nil, "%s: export type of sibling is %s but expected %s", fldPath.String(), siblingExport.Type, mapping.Type)
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

// IsRoot returns true if the current component is a root component
func (v *Validator) IsRoot() bool {
	return v.parent == nil
}
