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
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/datatype"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/landscapeconfig"
)

// NewValidator creates new import validator.
// It validates if all imports of a component are satisfied given a context.
func NewValidator(op installations.Operation, landscapeConfig *landscapeconfig.LandscapeConfig, parent *installations.Installation, siblings ...*installations.Installation) *Validator {
	return &Validator{
		Operation: op,

		lsConfig: landscapeConfig,
		parent:   parent,
		siblings: siblings,
	}
}

// Validate traverses through all components and validates if all imports are
// satisfied with the correct version
func (v *Validator) Validate(inst *installations.Installation) error {
	fldPath := field.NewPath(inst.Info.Name)
	for i, importMapping := range inst.Info.Spec.Imports {
		impPath := fldPath.Index(i)
		// check if the parent also imports my import
		err := v.checkImportMappingIsSatisfied(impPath, inst, importMapping)
		if err != nil {
			return err
		}
	}

	return nil
}

func (v *Validator) checkImportMappingIsSatisfied(fldPath *field.Path, inst *installations.Installation, mapping v1alpha1.DefinitionImportMapping) error {
	// check landscape config if I'm v root installation
	if v.IsRoot() {
		err := v.checkIfLandscapeConfigForMapping(fldPath, inst, mapping)
		if !installations.IsImportNotFoundError(err) {
			return err
		}
		if err == nil {
			return nil
		}
	} else {
		// check if the parent also imports my import
		err := v.checkIfParentHasImportForMapping(fldPath, inst, mapping)
		if !installations.IsImportNotFoundError(err) {
			return err
		}
		if err == nil {
			return nil
		}
	}

	// check if a sibling exports the given value
	err := v.checkIfSiblingsHaveImportForMapping(fldPath, inst, mapping)
	if !installations.IsImportNotFoundError(err) {
		return err
	}
	if err == nil {
		return nil
	}

	return installations.NewImportNotFoundError("No import found", nil)
}

func (v *Validator) checkIfLandscapeConfigForMapping(fldPath *field.Path, inst *installations.Installation, mapping v1alpha1.DefinitionImportMapping) error {
	importDef, err := inst.GetImportDefinition(mapping.To)
	if err != nil {
		return err
	}
	//importState, err := inst.ImportStatus().GetTo(mapping.To)
	//if err != nil {
	//	return err
	//}

	var val interface{}
	if err := v.lsConfig.Data.GetData(mapping.From, &val); err != nil {
		return installations.NewImportNotFoundErrorf(err, "%s: import in landscape config not found", fldPath.String())
	}

	//if importState.ConfigGeneration >= v.lsConfig.Info.Status.ConfigGeneration {
	//	return NewImportNotSatisfiedErrorf(nil, "%s: import has already run", fldPath.String())
	//}

	// validate types
	dt, ok := v.GetDataType(importDef.Type)
	if !ok {
		return installations.NewImportNotSatisfiedErrorf(nil, "%s: datatype %s cannot be found", fldPath.String(), importDef.Type)
	}
	if err := datatype.Validate(*dt, val); err != nil {
		return installations.NewImportNotSatisfiedErrorf(err, "%s: imported datatype does not have the expected type %s", fldPath.String(), importDef.Type)
	}

	return nil
}

func (v *Validator) checkIfParentHasImportForMapping(fldPath *field.Path, inst *installations.Installation, mapping v1alpha1.DefinitionImportMapping) error {
	importDef, err := inst.GetImportDefinition(mapping.To)
	if err != nil {
		return err
	}
	//importState, err := inst.ImportStatus().GetTo(mapping.To)
	//if err != nil {
	//	return err
	//}

	// check if the parent also imports my import
	parentImport, err := v.parent.GetImportDefinition(mapping.From)
	if err != nil {
		return installations.NewImportNotFoundErrorf(err, "%s: ImportDefinition not found", fldPath.String())
	}
	//parentImportState, err := v.parent.ImportStatus().GetTo(mapping.From)
	//if err != nil {
	//	return NewImportNotFoundErrorf(err, "%s: Import state not found", fldPath.String())
	//}

	// parent has to be progressing
	if v.parent.Info.Status.Phase != v1alpha1.ComponentPhaseProgressing {
		return installations.NewImportNotSatisfiedErrorf(nil, "%s: Parent has to be progressing to get imports", fldPath.String())
	}

	if parentImport.Type != importDef.Type {
		return installations.NewImportNotSatisfiedErrorf(nil, "%s: import type of parent is %s but expected %s", fldPath.String(), parentImport.Type, importDef.Type)
	}

	// check if the import of the parent is of v higher generation
	//if importState.ConfigGeneration >= parentImportState.ConfigGeneration {
	//	return NewImportNotSatisfiedErrorf(nil, "%s: import has already run", fldPath.String())
	//}

	return nil
}

func (v *Validator) checkIfSiblingsHaveImportForMapping(fldPath *field.Path, inst *installations.Installation, mapping v1alpha1.DefinitionImportMapping) error {

	for _, sibling := range v.siblings {
		sPath := fldPath.Child(sibling.Info.Name)
		err := v.checkIfSiblingHasImportForMapping(sPath, inst, mapping, sibling)
		if !installations.IsImportNotFoundError(err) {
			return err
		}
		if err == nil {
			return nil
		}
	}

	return installations.NewImportNotFoundErrorf(nil, "%s: no sibling installation found to satisfy the mapping", fldPath.String())
}

func (v *Validator) checkIfSiblingHasImportForMapping(fldPath *field.Path, inst *installations.Installation, mapping v1alpha1.DefinitionImportMapping, sibling *installations.Installation) error {
	importDef, err := inst.GetImportDefinition(mapping.To)
	if err != nil {
		return err
	}
	//importState, err := inst.ImportStatus().GetTo(mapping.To)
	//if err != nil {
	//	return err
	//}

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

	if sibling.Info.Status.Phase != v1alpha1.ComponentPhaseSucceeded {
		return installations.NewImportNotSatisfiedErrorf(nil, "%s: Sibling Installation has to be completed to get exports", fldPath.String())
	}

	if siblingExport.Type != importDef.Type {
		return installations.NewImportNotSatisfiedErrorf(nil, "%s: export type of sibling is %s but expected %s", fldPath.String(), siblingExport.Type, importDef.Type)
	}

	// check if the import of the parent is of v higher generation
	//if importState.ConfigGeneration >= sibling.Info.Status.ConfigGeneration {
	//	return NewImportNotSatisfiedErrorf(nil, "%s: import has already run", fldPath.String())
	//}

	// todo: check generation of other components in the dependency tree
	// we expect the parent's exported config generation to equal the highest among all subcomponents

	if sibling.Info.Status.ConfigGeneration < v.parent.Info.Status.ConfigGeneration {
		return installations.NewImportNotSatisfiedError("parent has higher generation than the imported configuration", nil)
	}

	return nil
}

// IsRoot returns true if the current component is a root component
func (v *Validator) IsRoot() bool {
	return v.parent == nil
}
