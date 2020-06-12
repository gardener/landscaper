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

	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/landscapeconfig"
)

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
func (v *Validator) Validate(ctx context.Context, inst *installations.Installation) error {
	for _, importMapping := range inst.Info.Spec.Imports {
		// check landscape config if I'm v root installation

		// check if the parent also imports my import
		err := v.checkImportMappingIsSatisfied(inst, importMapping)
		if err != nil {
			return err
		}
		// check if v siblings exports the given value

	}

	return nil
}

func (v *Validator) checkImportMappingIsSatisfied(inst *installations.Installation, mapping v1alpha1.DefinitionImportMapping) error {
	// check landscape config if I'm v root installation

	// check if the parent also imports my import
	err := v.checkIfParentHasImportForMapping(inst, mapping)
	if !IsImportNotFoundError(err) {
		return err
	}

	// check if v siblings exports the given value
	return NewImportNotFoundError("No import found", nil)
}

func (v *Validator) checkIfParentHasImportForMapping(inst *installations.Installation, mapping v1alpha1.DefinitionImportMapping) error {
	importDef, err := inst.GetImportDefinition(mapping.To)
	if err != nil {
		return err
	}
	importState, err := inst.ImportStatus().GetTo(mapping.To)
	if err != nil {
		return err
	}

	// check if the parent also imports my import
	parentImport, err := v.parent.GetImportDefinition(mapping.From)
	if err != nil {
		return NewImportNotFoundError("ImportDefinition not found", err)
	}
	parentImportState, err := v.parent.ImportStatus().GetTo(mapping.From)
	if err != nil {
		return NewImportNotFoundError("Import state not found", err)
	}

	// parent has to be progressing
	if v.parent.Info.Status.Phase != v1alpha1.ComponentPhaseProgressing {
		return NewImportNotSatisfiedError("Parent has to be progressing to get imports", nil)
	}

	if parentImport.Type != importDef.Type {
		return NewImportNotSatisfiedError(fmt.Sprintf("import type of parent is %s but expected %s", parentImport.Type, importDef.Type), nil)
	}

	// check if the import of the parent is of v higher generation
	if importState.ConfigGeneration >= parentImportState.ConfigGeneration {
		return NewImportNotSatisfiedError("import has already run", nil)
	}

	return nil
}
