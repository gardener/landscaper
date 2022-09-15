// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"fmt"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
)

// Installation is the internal representation of a installation
type InstallationImportsAndBlueprint struct {
	InstallationAndImports
	blueprint *blueprints.Blueprint
}

// New creates a new internal representation of an installation with blueprint
func NewInstallationImportsAndBlueprint(inst *lsv1alpha1.Installation, blueprint *blueprints.Blueprint) *InstallationImportsAndBlueprint {
	internalInst := &InstallationImportsAndBlueprint{
		InstallationAndImports: *NewInstallationAndImports(inst),
		blueprint:              blueprint,
	}

	return internalInst
}

func (i *InstallationImportsAndBlueprint) GetBlueprint() *blueprints.Blueprint {
	return i.blueprint
}

// GetImportDefinition return the import for a given key
func (i *InstallationImportsAndBlueprint) GetImportDefinition(key string) (lsv1alpha1.ImportDefinition, error) {
	imports := i.getFlattenedImports(i.blueprint.Info.Imports)
	for _, def := range imports {
		if def.Name == key {
			return def, nil
		}
	}
	return lsv1alpha1.ImportDefinition{}, fmt.Errorf("import with key %s not found", key)
}

// GetExportDefinition return the export definition for a given key
func (i *InstallationImportsAndBlueprint) GetExportDefinition(key string) (lsv1alpha1.ExportDefinition, error) {
	for _, def := range i.blueprint.Info.Exports {
		if def.Name == key {
			return def, nil
		}
	}
	return lsv1alpha1.ExportDefinition{}, fmt.Errorf("export with key %s not found", key)
}

// getFlattenedImports is an auxiliary method that flattens the tree of conditional imports into a list
func (i *InstallationImportsAndBlueprint) getFlattenedImports(importList lsv1alpha1.ImportDefinitionList) lsv1alpha1.ImportDefinitionList {
	res := lsv1alpha1.ImportDefinitionList{}
	for _, def := range importList {
		res = append(res, def)
		if def.ConditionalImports != nil && len(def.ConditionalImports) > 0 {
			res = append(res, i.getFlattenedImports(def.ConditionalImports)...)
		}
	}
	return res
}
