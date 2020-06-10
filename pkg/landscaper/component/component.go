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

package component

import (
	"fmt"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// Installation is the internal representation of a installation
type Installation struct {
	Info       *lsv1alpha1.ComponentInstallation
	Definition *lsv1alpha1.ComponentDefinition

	imports map[string]lsv1alpha1.DefinitionImport
	exports map[string]lsv1alpha1.DefinitionExport

	// indexes the import state with from/to as key
	importsStatus ImportStatus
}

// New creates a new internal representation of an installation
func New(inst *lsv1alpha1.ComponentInstallation, def *lsv1alpha1.ComponentDefinition) (*Installation, error) {

	internalInst := &Installation{
		Info:       inst,
		Definition: def,

		imports: make(map[string]lsv1alpha1.DefinitionImport, len(def.Imports)),
		exports: make(map[string]lsv1alpha1.DefinitionExport, len(def.Exports)),

		importsStatus: ImportStatus{
			From: make(map[string]*lsv1alpha1.ImportState, len(inst.Status.Imports)),
			To:   make(map[string]*lsv1alpha1.ImportState, len(inst.Status.Imports)),
		},
	}

	for _, importDef := range def.Imports {
		internalInst.imports[importDef.Key] = importDef
	}
	for _, exportDef := range def.Exports {
		internalInst.exports[exportDef.Key] = exportDef
	}

	for _, importStatus := range inst.Status.Imports {
		internalInst.importsStatus.add(importStatus)
	}

	return internalInst, nil
}

// ImportStatus returns the internal representation of the internal import state
func (i *Installation) ImportStatus() *ImportStatus {
	return &i.importsStatus
}

// GetImportDefinition return the import for a given key
func (i *Installation) GetImportDefinition(key string) (lsv1alpha1.DefinitionImport, error) {
	def, ok := i.imports[key]
	if !ok {
		return lsv1alpha1.DefinitionImport{}, fmt.Errorf("import with key %s not found", key)
	}
	return def, nil
}

// GetExportDefinition return the export definition for a given key
func (i *Installation) GetExportDefinition(key string) (lsv1alpha1.DefinitionExport, error) {
	def, ok := i.exports[key]
	if !ok {
		return lsv1alpha1.DefinitionExport{}, fmt.Errorf("export with key %s not found", key)
	}
	return def, nil
}
