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

	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/component"
)

// importsAreSatisfied traverses through all components and validates if all imports are
// satisfied with the correct version
func (a *actuator) importsAreSatisfied(ctx context.Context, landscapeConfig *v1alpha1.LandscapeConfiguration, inst *component.Installation, lsCtx *Context) (bool, error) {
	var (
		parent = lsCtx.Parent
		//siblings = lsCtx.Siblings
	)

	for _, importMapping := range inst.Info.Spec.Imports {
		// check landscape config if I'm a root installation

		// check if the parent also imports my import
		ok, err := a.checkIfParentHasImportForMapping(inst, parent, importMapping)
		if err != nil {
			return false, err
		}
		if ok {
			continue
		}
		// check if a siblings exports the given value

	}

	return true, nil
}

func (a *actuator) checkIfParentHasImportForMapping(inst *component.Installation, parent *component.Installation, mapping v1alpha1.DefinitionImportMapping) (bool, error) {
	importDef, err := inst.GetImportDefinition(mapping.To)
	if err != nil {
		return false, err
	}
	importState, err := inst.ImportStatus().GetTo(mapping.To)
	if err != nil {
		return false, err
	}

	// check if the parent also imports my import
	parentImport, err := parent.GetImportDefinition(mapping.From)
	if err != nil {
		return false, err
	}
	parentImportState, err := parent.ImportStatus().GetTo(mapping.From)
	if err != nil {
		return false, err
	}

	// parent has to be progressing
	if parent.Info.Status.Phase != v1alpha1.ComponentPhaseProgressing {
		return false, errors.New("Parent has to be progressing to get imports")
	}

	if parentImport.Type != importDef.Type {
		return false, errors.Errorf("import type of parent is %s but expected %s", parentImport.Type, importDef.Type)
	}

	// check if the import of the parent is of a higher generation
	if importState.ConfigGeneration >= parentImportState.ConfigGeneration {
		return false, errors.Errorf("import has already run")
	}

	return true, nil
}
