// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"os"

	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/mandelsoft/vfs/pkg/vfs"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/pkg/apis/core"
	"github.com/gardener/landscaper/pkg/kubernetes"
)

// ValidateBlueprint validates a Blueprint
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
	_, allErrs := validateBlueprintImportDefinitions(fldPath, imports, sets.NewString())
	return allErrs
}

// validateBlueprintImportDefinitions validates a list of import definitions
func validateBlueprintImportDefinitions(fldPath *field.Path, imports []core.ImportDefinition, importNames sets.String) (sets.String, field.ErrorList) {
	allErrs := field.ErrorList{}

	for i, importDef := range imports {
		defPath := fldPath.Index(i)
		if len(importDef.Name) != 0 {
			defPath = defPath.Key(importDef.Name)
			if importNames.Has(importDef.Name) {
				allErrs = append(allErrs, field.Duplicate(defPath, "duplicate import name"))
			}
			importNames.Insert(importDef.Name)
		}

		required := true
		if importDef.Required != nil {
			required = *importDef.Required
		}
		if len(importDef.ConditionalImports) > 0 && required {
			allErrs = append(allErrs, field.Invalid(defPath, importDef.Name, "conditional imports on required import"))
		}

		allErrs = append(allErrs, ValidateFieldValueDefinition(defPath, importDef.FieldValueDefinition)...)
		conditionalImportNames, tmpErrs := validateBlueprintImportDefinitions(defPath.Child("conditionalImports"), importDef.ConditionalImports, importNames)
		allErrs = append(allErrs, tmpErrs...)
		importNames.Insert(conditionalImportNames.UnsortedList()...)
	}

	return importNames, allErrs
}

// ValidateBlueprintExportDefinitions validates a list of export definitions
func ValidateBlueprintExportDefinitions(fldPath *field.Path, exports []core.ExportDefinition) field.ErrorList {
	allErrs := field.ErrorList{}

	exportNames := sets.NewString()
	for i, exportDef := range exports {
		defPath := fldPath.Index(i)
		if len(exportDef.Name) != 0 {
			defPath = defPath.Key(exportDef.Name)
		}

		allErrs = append(allErrs, ValidateFieldValueDefinition(defPath, exportDef.FieldValueDefinition)...)

		if len(exportDef.Name) != 0 && exportNames.Has(exportDef.Name) {
			allErrs = append(allErrs, field.Duplicate(defPath, "duplicated export name"))
		}
		exportNames.Insert(exportDef.Name)
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
		execPath := fldPath.Index(i)
		if len(exec.Name) == 0 {
			allErrs = append(allErrs, field.Required(execPath.Child("name"), "name must be defined"))
		} else {
			execPath = execPath.Key(exec.Name)
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

		// need to reduce lenght by 1 because "-" is added during subinstallation creation
		if len(template.Name) > InstallationGenerateNameMaxLength-1 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("name"), template.Name, validation.MaxLenError(InstallationGenerateNameMaxLength-1)))
		}
	}

	if len(template.Blueprint.Ref) == 0 && len(template.Blueprint.Filesystem) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("blueprint"), "a blueprint must be defined"))
	}

	allErrs = append(allErrs, ValidateInstallationTemplateImports(template.Imports, fldPath.Child("imports"))...)
	allErrs = append(allErrs, ValidateInstallationExports(template.Exports, fldPath.Child("exports"))...)

	return allErrs
}

// ValidateInstallationTemplateImports validates the imports of an InstallationTemplate
func ValidateInstallationTemplateImports(imports core.InstallationImports, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, ValidateInstallationTemplateDataImports(imports.Data, fldPath.Child("data"))...)
	allErrs = append(allErrs, ValidateInstallationTargetImports(imports.Targets, fldPath.Child("targets"))...)

	return allErrs
}

// ValidateInstallationTemplateDataImports validates the data imports of an InstallationTemplate
func ValidateInstallationTemplateDataImports(imports []core.DataImport, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	importNames := map[string]bool{}
	for idx, imp := range imports {
		impPath := fldPath.Index(idx)

		if imp.DataRef == "" {
			allErrs = append(allErrs, field.Required(impPath.Child("dataRef"), "dataRef must not be empty"))
		}

		if imp.SecretRef != nil {
			allErrs = append(allErrs, field.Forbidden(impPath.Child("secretRef"), "secret references are not allowed in a installation template"))
		}
		if imp.ConfigMapRef != nil {
			allErrs = append(allErrs, field.Forbidden(impPath.Child("configMapRef"), "configMap references are not allowed in a installation template"))
		}

		if imp.Name == "" {
			allErrs = append(allErrs, field.Required(impPath.Child("name"), "name must not be empty"))
			continue
		}
		if importNames[imp.Name] {
			allErrs = append(allErrs, field.Duplicate(fldPath.Index(idx), imp.Name))
		}
		importNames[imp.Name] = true
	}

	return allErrs
}
