// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package validation

import (
	"os"

	"github.com/mandelsoft/vfs/pkg/vfs"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/pkg/apis/core"
	"github.com/gardener/landscaper/pkg/kubernetes"
)

// ValidateInstallation validates an Installation
func ValidateBlueprint(fs vfs.FileSystem, blueprint *core.Blueprint) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateBlueprintImportDefinitions(field.NewPath("imports"), blueprint.Imports)...)
	allErrs = append(allErrs, ValidateBlueprintExportDefinitions(field.NewPath("exports"), blueprint.Exports)...)
	allErrs = append(allErrs, ValidateTemplateExecutorList(field.NewPath("deployExecutors"), blueprint.DeployExecutions)...)
	allErrs = append(allErrs, ValidateTemplateExecutorList(field.NewPath("exportExecutors"), blueprint.ExportExecutions)...)
	allErrs = append(allErrs, ValidateSubinstallations(field.NewPath("subinstallations"), fs, blueprint.Imports, blueprint.Subinstallations)...)
	return allErrs
}

// ValidateBlueprintImportDefinitions validates a list of import definitions
func ValidateBlueprintImportDefinitions(fldPath *field.Path, imports []core.ImportDefinition) field.ErrorList {
	allErrs := field.ErrorList{}

	importNames := sets.NewString()
	for i, importDef := range imports {
		defPath := fldPath.Child(importDef.Name)
		if len(importDef.Name) == 0 {
			defPath = fldPath.Index(i)
		}

		allErrs = append(allErrs, ValidateFieldValueDefinition(defPath, importDef.FieldValueDefinition)...)
		allErrs = append(allErrs, ValidateBlueprintImportDefinitions(defPath.Child("conditionalImports"), importDef.ConditionalImports)...)

		if len(importDef.Name) != 0 && importNames.Has(importDef.Name) {
			allErrs = append(allErrs, field.Duplicate(defPath, "duplicated import name"))
		}
		importNames.Insert(importDef.Name)
	}

	return allErrs
}

// ValidateBlueprintExportDefinitions validates a list of export definitions
func ValidateBlueprintExportDefinitions(fldPath *field.Path, exports []core.ExportDefinition) field.ErrorList {
	allErrs := field.ErrorList{}

	exportNames := sets.NewString()
	for i, importDef := range exports {
		defPath := fldPath.Child(importDef.Name)
		if len(importDef.Name) == 0 {
			defPath = fldPath.Index(i)
		}

		allErrs = append(allErrs, ValidateFieldValueDefinition(defPath, importDef.FieldValueDefinition)...)

		if len(importDef.Name) != 0 && exportNames.Has(importDef.Name) {
			allErrs = append(allErrs, field.Duplicate(defPath, "duplicated export name"))
		}
		exportNames.Insert(importDef.Name)
	}

	return allErrs
}

// ValidateFieldValueDefinition validates a field value definition
func ValidateFieldValueDefinition(fldPath *field.Path, def core.FieldValueDefinition) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(def.Name) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "name must not be empty"))
	}
	if len(def.Schema) == 0 && len(def.TargetType) == 0 {
		allErrs = append(allErrs, field.Required(fldPath, "schema or targetType must not be empty"))
	}

	if len(def.Schema) != 0 {
		allErrs = append(allErrs, ValidateJsonSchema(fldPath, def.Schema)...)
	}

	return allErrs
}

// ValidateJsonSchema validates a json schema
func ValidateJsonSchema(fldPath *field.Path, schema core.JSONSchemaDefinition) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateTemplateExecutorList validates a list of template executors
func ValidateTemplateExecutorList(fldPath *field.Path, list []core.TemplateExecutor) field.ErrorList {
	allErrs := field.ErrorList{}
	names := sets.NewString()
	for i, exec := range list {
		execPath := fldPath.Child(exec.Name)
		if len(exec.Name) == 0 {
			execPath = fldPath.Index(i)
			allErrs = append(allErrs, field.Required(execPath.Child("name"), "name must be defined"))
		}

		if len(exec.Type) == 0 {
			allErrs = append(allErrs, field.Required(execPath.Child("type"), "type must be defined"))
		}

		if len(exec.Name) != 0 && names.Has(exec.Name) {
			allErrs = append(allErrs, field.Duplicate(execPath, "duplicated executor name"))
		}
		names.Insert(exec.Name)
	}
	return allErrs
}

// ValidateSubinstallations validates all inline subinstallation and installation templates from a file
func ValidateSubinstallations(fldPath *field.Path, fs vfs.FileSystem, blueprintImportDefs []core.ImportDefinition, subinstallations []core.SubinstallationTemplate) field.ErrorList {
	var (
		allErrs             = field.ErrorList{}
		names               = sets.NewString()
		importedDataObjects = make([]Import, 0)
		exportedDataObjects = sets.NewString()
		importedTargets     = make([]Import, 0)
		exportedTargets     = sets.NewString()

		blueprintDataImports   = sets.NewString()
		blueprintTargetImports = sets.NewString()
	)

	for _, bImport := range blueprintImportDefs {
		if len(bImport.Schema) != 0 {
			blueprintDataImports.Insert(bImport.Name)
		} else if len(bImport.TargetType) != 0 {
			blueprintTargetImports.Insert(bImport.Name)
		}

	}

	for i, subinst := range subinstallations {
		instPath := fldPath.Index(i)

		if subinst.InstallationTemplate == nil && len(subinst.File) == 0 {
			allErrs = append(allErrs, field.Required(instPath, "subinstallation has to be defined inline or by file"))
			continue
		}
		if subinst.InstallationTemplate != nil && len(subinst.File) != 0 {
			allErrs = append(allErrs, field.Invalid(instPath, subinst, "subinstallation must not be defined inline and by file"))
			continue
		}

		instTmpl := subinst.InstallationTemplate
		if len(subinst.File) != 0 {
			data, err := vfs.ReadFile(fs, subinst.File)
			if err != nil {
				if os.IsNotExist(err) {
					allErrs = append(allErrs, field.NotFound(instPath.Child("file"), subinst.File))
					continue
				}
				allErrs = append(allErrs, field.InternalError(instPath.Child("file"), err))
				continue
			}

			instTmpl = &core.InstallationTemplate{}
			if _, _, err := serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDecoder().Decode(data, nil, instTmpl); err != nil {
				allErrs = append(allErrs, field.Invalid(instPath.Child("file"), subinst.File, err.Error()))
				continue
			}
		}

		for i, do := range instTmpl.Exports.Data {
			dataPath := instPath.Child("exports").Child("data").Index(i).Key(do.Name)
			if exportedDataObjects.Has(do.DataRef) {
				allErrs = append(allErrs, field.Forbidden(dataPath, "export is already exported by another installation"))
			}
			if blueprintDataImports.Has(do.DataRef) {
				allErrs = append(allErrs, field.Forbidden(dataPath, "export is imported by its parent"))
			}
			exportedDataObjects.Insert(do.DataRef)
		}
		for i, do := range instTmpl.Imports.Data {
			importedDataObjects = append(importedDataObjects, Import{
				Name: do.DataRef,
				Path: instPath.Child("imports").Child("data").Index(i).Key(do.Name),
			})
		}
		for i, target := range instTmpl.Exports.Targets {
			targetPath := instPath.Child("exports").Child("targets").Index(i).Key(target.Name)
			if exportedTargets.Has(target.Target) {
				allErrs = append(allErrs, field.Forbidden(targetPath, "export is already exported by another installation"))
			}
			if blueprintTargetImports.Has(target.Target) {
				allErrs = append(allErrs, field.Forbidden(targetPath, "export is imported by its parent"))
			}
			exportedTargets.Insert(target.Target)
		}
		for i, target := range instTmpl.Imports.Targets {
			importedTargets = append(importedTargets, Import{
				Name: target.Target,
				Path: instPath.Child("imports").Child("targets").Index(i).Key(target.Name),
			})
		}

		allErrs = append(allErrs, ValidateInstallationTemplate(instPath, instTmpl)...)
		if len(instTmpl.Name) != 0 && names.Has(instTmpl.Name) {
			allErrs = append(allErrs, field.Duplicate(instPath, "duplicated subinstallation"))
		}
		names.Insert(instTmpl.Name)
	}

	// validate that all imported values are either satisfied by the blueprint or by another sibling
	allErrs = append(allErrs, ValidateSatisfiedImports(blueprintDataImports, exportedDataObjects, importedDataObjects)...)
	allErrs = append(allErrs, ValidateSatisfiedImports(blueprintTargetImports, exportedTargets, importedTargets)...)

	return allErrs
}

// Import defines a internal import struct for validation.
type Import struct {
	Name string
	Path *field.Path
}

// ValidateSatisfiedImports validates that all imported data is satisfied.
func ValidateSatisfiedImports(blueprintImports, exports sets.String, imports []Import) field.ErrorList {
	allErrs := field.ErrorList{}
	for _, dataImport := range imports {
		if !exports.Has(dataImport.Name) && !blueprintImports.Has(dataImport.Name) {
			allErrs = append(allErrs, field.NotFound(dataImport.Path, "import not satisfied"))
		}
	}
	return allErrs
}

// ValidateInstallationTemplate validates a installation template
func ValidateInstallationTemplate(fldPath *field.Path, template *core.InstallationTemplate) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(template.Name) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "name must be defined"))
	} else {
		for _, msg := range apivalidation.NameIsDNSLabel(template.Name, false) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("name"), template.Name, msg))
		}
	}

	if len(template.Blueprint.Ref) == 0 && len(template.Blueprint.Filesystem) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("blueprint"), "a blueprint must be defined"))
	}

	allErrs = append(allErrs, ValidateInstallationImports(template.Imports, fldPath.Child("imports"))...)
	allErrs = append(allErrs, ValidateInstallationExports(template.Exports, fldPath.Child("exports"))...)

	return allErrs
}
