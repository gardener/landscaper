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
	"fmt"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// Installation is the internal representation of a installation
type Installation struct {
	Info       *lsv1alpha1.Installation
	Definition *lsv1alpha1.Blueprint

	imports map[string]lsv1alpha1.ImportDefinition
	exports map[string]lsv1alpha1.ExportDefinition

	// indexes the import state with from/to as key
	importsStatus ImportStatus
}

// ImportMapping is the internal representation of a import mapping and its defintion
type ImportMapping struct {
	lsv1alpha1.DefinitionImportMapping
	lsv1alpha1.ImportDefinition
}

// ExportMapping is the internal representation of a export mapping and its defintion
type ExportMapping struct {
	lsv1alpha1.DefinitionExportMapping
	lsv1alpha1.ExportDefinition
}

// New creates a new internal representation of an installation
func New(inst *lsv1alpha1.Installation, def *lsv1alpha1.Blueprint) (*Installation, error) {
	internalInst := &Installation{
		Info:       inst,
		Definition: def,

		imports: make(map[string]lsv1alpha1.ImportDefinition, len(def.Imports)),
		exports: make(map[string]lsv1alpha1.ExportDefinition, len(def.Exports)),

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
		internalInst.importsStatus.set(importStatus)
	}

	return internalInst, nil
}

// ImportStatus returns the internal representation of the internal import state
func (i *Installation) ImportStatus() *ImportStatus {
	return &i.importsStatus
}

// GetImportDefinition return the import for a given key
func (i *Installation) GetImportDefinition(key string) (lsv1alpha1.ImportDefinition, error) {
	def, ok := i.imports[key]
	if !ok {
		return lsv1alpha1.ImportDefinition{}, fmt.Errorf("import with key %s not found", key)
	}
	return def, nil
}

// GetImportMappings returns all import mappings of a installation.
// If a import definition is not defined in the mappings it will be automatically added with the default mappings.
// todo: check for unused mappings.
func (i *Installation) GetImportMappings() ([]ImportMapping, error) {
	mappings := make([]ImportMapping, 0)
	for _, obj := range i.Definition.Imports {
		def := obj.DeepCopy()
		// try to get a import mapping
		mapping, err := i.GetImportMappingTo(def.Key)
		if err == nil {
			mappings = append(mappings, ImportMapping{
				ImportDefinition:        *def,
				DefinitionImportMapping: mapping,
			})
			continue
		}
		mappings = append(mappings, ImportMapping{
			ImportDefinition: *def,
			DefinitionImportMapping: lsv1alpha1.DefinitionImportMapping{
				DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{
					From: def.Key,
					To:   def.Key,
				},
			},
		})
	}
	return mappings, nil
}

// GetExportMappings returns all exported mappings of a installation.
// If a export definition is not defined in the mappings it will be automatically added with the default mappings.
// todo: check for unused mappings.
func (i *Installation) GetExportMappings() ([]ExportMapping, error) {
	mappings := make([]ExportMapping, 0)
	for _, obj := range i.Definition.Exports {
		def := obj.DeepCopy()
		// try to get a import mapping
		mapping, err := i.GetExportMappingTo(def.Key)
		if err == nil {
			mappings = append(mappings, ExportMapping{
				ExportDefinition:        *def,
				DefinitionExportMapping: mapping,
			})
			continue
		}
		mappings = append(mappings, ExportMapping{
			ExportDefinition: *def,
			DefinitionExportMapping: lsv1alpha1.DefinitionExportMapping{
				DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{
					From: def.Key,
					To:   def.Key,
				},
			},
		})
	}
	return mappings, nil
}

// GetImportMappingFrom returns the import mapping of the installation that imports data from the given key
func (i *Installation) GetImportMappingFrom(key string) (lsv1alpha1.DefinitionImportMapping, error) {
	for _, mapping := range i.Info.Spec.Imports {
		if mapping.From == key {
			return mapping, nil
		}
	}

	return lsv1alpha1.DefinitionImportMapping{}, fmt.Errorf("import mapping for key %s not found", key)
}

// GetImportMappingTo returns the import mapping of the installation that imports data to the given key
func (i *Installation) GetImportMappingTo(key string) (lsv1alpha1.DefinitionImportMapping, error) {
	for _, mapping := range i.Info.Spec.Imports {
		if mapping.To == key {
			return mapping, nil
		}
	}

	return lsv1alpha1.DefinitionImportMapping{}, fmt.Errorf("import mapping for key %s not found", key)
}

// GetExportDefinition return the export definition for a given key
func (i *Installation) GetExportDefinition(key string) (lsv1alpha1.ExportDefinition, error) {
	def, ok := i.exports[key]
	if !ok {
		return lsv1alpha1.ExportDefinition{}, fmt.Errorf("export with key %s not found", key)
	}
	return def, nil
}

// GetExportMappingTo returns the export mapping of the installation that exports to the given key
func (i *Installation) GetExportMappingTo(key string) (lsv1alpha1.DefinitionExportMapping, error) {
	for _, mapping := range i.Info.Spec.Exports {
		if mapping.To == key {
			return mapping, nil
		}
	}

	return lsv1alpha1.DefinitionExportMapping{}, fmt.Errorf("export mapping for key %s not found", key)
}

// GetExportMappingFrom returns the export mapping of the installation that exports from the given key
func (i *Installation) GetExportMappingFrom(key string) (lsv1alpha1.DefinitionExportMapping, error) {
	for _, mapping := range i.Info.Spec.Exports {
		if mapping.From == key {
			return mapping, nil
		}
	}

	return lsv1alpha1.DefinitionExportMapping{}, fmt.Errorf("export mapping for key %s not found", key)
}
