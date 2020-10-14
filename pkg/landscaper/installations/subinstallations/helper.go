// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package subinstallations

import (
	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
)

//// AddDefaultImports adds all default mappings of im and exports if they are not already defined
//func AddDefaultImports(inst *lsv1alpha1.Installation, blueprint *lsv1alpha1.Blueprint) {
//	dataImports := sets.NewString()
//	targetImports := sets.NewString()
//	for _, dataImport := range inst.Spec.Imports.Data {
//		dataImports.Insert(dataImport.Name)
//	}
//	for _, targetImport := range inst.Spec.Imports.Targets {
//		targetImports.Insert(targetImport.Name)
//	}
//	for _, importDef := range blueprint.Imports {
//		if !dataImports.Has(importDef.Name) {
//			inst.Spec.Imports = append(inst.Spec.Imports, lsv1alpha1.DefinitionImportMapping{
//				DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{
//					From: importDef.Name,
//					To:   importDef.Name,
//				},
//			})
//		}
//	}
//
//	dataImports = sets.NewString()
//	for _, mapping := range inst.Spec.Exports {
//		dataImports.Insert(mapping.From)
//	}
//	for _, importDef := range blueprint.Exports {
//		if !dataImports.Has(importDef.Name) {
//			inst.Spec.Exports = append(inst.Spec.Exports, lsv1alpha1.DefinitionExportMapping{
//				DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{
//					From: importDef.Name,
//					To:   importDef.Name,
//				},
//			})
//		}
//	}
//}

//// installationNeedsUpdate check if a definition reference has been updated
//func installationNeedsUpdate(def lsv1alpha1.BlueprintReferenceTemplate, inst *lsv1alpha1.Installation) bool {
//	// check if the reference itself has changed
//	if def.Info != inst.Spec.Blueprint {
//		return true
//	}
//
//	for _, mapping := range def.Imports {
//		if !hasMappingOfImports(mapping, inst.Spec.Imports) {
//			return true
//		}
//	}
//
//	for _, mapping := range def.Exports {
//		if !hasMappingOfExports(mapping, inst.Spec.Exports) {
//			return true
//		}
//	}
//
//	if len(inst.Spec.Imports) != len(def.Imports) {
//		return true
//	}
//
//	if len(inst.Spec.Exports) != len(def.Exports) {
//		return true
//	}
//
//	return false
//}
//
//func hasMappingOfImports(search lsv1alpha1.DefinitionImportMapping, mappings []lsv1alpha1.DefinitionImportMapping) bool {
//	for _, mapping := range mappings {
//		if mapping.To == search.To && mapping.From == search.From {
//			return true
//		}
//	}
//	return false
//}
//
//func hasMappingOfExports(search lsv1alpha1.DefinitionExportMapping, mappings []lsv1alpha1.DefinitionExportMapping) bool {
//	for _, mapping := range mappings {
//		if mapping.To == search.To && mapping.From == search.From {
//			return true
//		}
//	}
//	return false
//}

// getDefinitionReference returns the definition reference by name
func getDefinitionReference(blueprint *blueprints.Blueprint, name string) (*lsv1alpha1.InstallationTemplate, bool) {
	for _, ref := range blueprint.Subinstallations {
		if ref.Name == name {
			return ref, true
		}
	}
	return nil, false
}
