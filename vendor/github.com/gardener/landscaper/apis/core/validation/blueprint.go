// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"

	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/apis/core"
	coreinstall "github.com/gardener/landscaper/apis/core/install"
)

var landscaperScheme = runtime.NewScheme()
var IsIndexedRegex = regexp.MustCompile(`(?P<imp>.*)\[(?P<idx>[1-9]?[0-9]*)\]$`)

func init() {
	coreinstall.Install(landscaperScheme)
	relevantConfigFields = computeRelevantConfigFields()
}

// ValidateBlueprint validates a Blueprint
func ValidateBlueprint(blueprint *core.Blueprint) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateBlueprintImportDefinitions(field.NewPath("imports"), blueprint.Imports)...)
	allErrs = append(allErrs, ValidateBlueprintExportDefinitions(field.NewPath("exports"), blueprint.Exports)...)
	allErrs = append(allErrs, ValidateTemplateExecutorList(field.NewPath("deployExecutions"), blueprint.DeployExecutions)...)
	allErrs = append(allErrs, ValidateTemplateExecutorList(field.NewPath("exportExecutions"), blueprint.ExportExecutions)...)
	allErrs = append(allErrs, ValidateSubinstallations(field.NewPath("subinstallations"), blueprint.Subinstallations)...)
	allErrs = append(allErrs, ValidateTemplateExecutorList(field.NewPath("subinstallationExecutions"), blueprint.SubinstallationExecutions)...)
	return allErrs
}

// ValidateBlueprintWithInstallationTemplates validates a Blueprint
func ValidateBlueprintWithInstallationTemplates(blueprint *core.Blueprint, installationTemplates []*core.InstallationTemplate) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateBlueprint(blueprint)...)
	allErrs = append(allErrs, ValidateInstallationTemplates(field.NewPath(""), blueprint.Imports, installationTemplates)...)
	return allErrs
}

// ValidateBlueprintImportDefinitions validates a list of import definitions
func ValidateBlueprintImportDefinitions(fldPath *field.Path, imports []core.ImportDefinition) field.ErrorList {
	_, allErrs := validateBlueprintImportDefinitions(fldPath, imports, sets.NewString())
	return allErrs
}

// validateBlueprintImportDefinitions validates a list of import definitions
func validateBlueprintImportDefinitions(fldPath *field.Path, imports []core.ImportDefinition, importNames sets.String) (sets.String, field.ErrorList) { //nolint:staticcheck // Ignore SA1019 // TODO: change to generic set
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

		if len(importDef.Type) != 0 {
			// type is specified, use new validation
			expectedConfigs, ok := importTypesWithExpectedConfig[string(importDef.Type)]
			if ok {
				// valid type, check that the required config is given
				allErrs = append(allErrs, validateMutuallyExclusiveConfig(defPath, importDef, expectedConfigs, string(importDef.Type))...)
			} else {
				// specified type is not among the valid types
				allErrs = append(allErrs, field.NotSupported(defPath.Child("type"), string(importDef.Type), keys(importTypesWithExpectedConfig)))
			}
		} else {
			// type is not specified, fallback to validation based on specified fields
			if importDef.Schema == nil && len(importDef.TargetType) == 0 {
				allErrs = append(allErrs, field.Required(defPath, "either schema or targetType must not be empty"))
			}
			if importDef.Schema != nil && len(importDef.TargetType) != 0 {
				allErrs = append(allErrs, field.Invalid(defPath, importDef, "either schema or targetType must be specified, not both"))
			}
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

		if len(exportDef.Type) != 0 {
			// type is specified, use new validation
			expectedConfigs, ok := exportTypesWithExpectedConfig[string(exportDef.Type)]
			if ok {
				// valid type, check that the required config is given
				allErrs = append(allErrs, validateMutuallyExclusiveConfig(defPath, exportDef, expectedConfigs, string(exportDef.Type))...)
			} else {
				// specified type is not among the valid types
				allErrs = append(allErrs, field.NotSupported(defPath.Child("type"), string(exportDef.Type), keys(importTypesWithExpectedConfig)))
			}
		} else {
			// type is not specified, fallback to validation based on specified fields
			allErrs = append(allErrs, ValidateExactlyOneOf(defPath, exportDef, "Schema", "TargetType")...)
		}

	}

	return allErrs
}

// validateMutuallyExclusiveConfig validates that the expected configs for the import/export definition are set and that no other config is set
// In this context, a 'config is set' if it
//   - can be nil but is not
//   - is not the zero value if it cannot be nil
func validateMutuallyExclusiveConfig(fldPath *field.Path, def interface{}, expectedConfigs []string, defType string) field.ErrorList {
	allErrs := field.ErrorList{}
	val := reflect.ValueOf(def)
	for key := range relevantConfigFields {
		var f reflect.Value
		if _, ok := isFieldValueDefinition[key]; ok {
			f = val.FieldByName("FieldValueDefinition")
		} else {
			f = val
		}
		f = f.FieldByName(key)
		if !f.IsValid() {
			// definition doesn't have this field
			// this can happen because import and export definitions support different fields
			continue
		}
		kind := f.Kind()
		expected := false
		for _, exp := range expectedConfigs {
			if key == exp {
				expected = true
				break
			}
		}
		if expected {
			if ((kind == reflect.Ptr || kind == reflect.Slice || kind == reflect.Map || kind == reflect.Interface) && f.IsNil()) || f.IsZero() {
				// field should be set but is empty
				allErrs = append(allErrs, field.Required(fldPath, fmt.Sprintf("%s must not be empty for type %s", key, defType)))
			}
		} else {
			if ((kind == reflect.Ptr || kind == reflect.Slice || kind == reflect.Map || kind == reflect.Interface) && !f.IsNil()) || !f.IsZero() {
				// field should not be set but it is
				allErrs = append(allErrs, field.Invalid(fldPath, def, fmt.Sprintf("unexpected config '%s', only [%s] should be set", key, strings.Join(expectedConfigs, ", "))))
			}
		}
	}
	return allErrs
}

// ValidateFieldValueDefinition validates a field value definition
func ValidateFieldValueDefinition(fldPath *field.Path, def core.FieldValueDefinition) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(def.Name) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "name must not be empty"))
	}
	if def.Schema != nil {
		allErrs = append(allErrs, ValidateJsonSchema(fldPath, def.Schema)...)
	}

	return allErrs
}

// ValidateJsonSchema validates a json schema
func ValidateJsonSchema(fldPath *field.Path, schema *core.JSONSchemaDefinition) field.ErrorList {
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
func ValidateSubinstallations(fldPath *field.Path, subinstallations []core.SubinstallationTemplate) field.ErrorList {
	allErrs := field.ErrorList{}

	for i, subinst := range subinstallations {
		instPath := fldPath.Index(i)

		errs := ValidateExactlyOneOf(instPath, subinst, "File", "InstallationTemplate")
		allErrs = append(allErrs, errs...)
		if len(errs) != 0 {
			continue
		}

		if subinst.InstallationTemplate != nil {
			allErrs = append(allErrs, ValidateInstallationTemplate(instPath, subinst.InstallationTemplate)...)
		}
	}
	return allErrs
}

// ValidateInstallationTemplates validates a list of subinstallations.
// Take care to also include all templated templates for proper validation.
func ValidateInstallationTemplates(fldPath *field.Path, blueprintImportDefs []core.ImportDefinition, subinstallations []*core.InstallationTemplate) field.ErrorList {
	var (
		allErrs             = field.ErrorList{}
		names               = sets.NewString()
		importedDataObjects = make([]Import, 0)
		exportedDataObjects = map[string]string{}
		importedTargets     = make([]Import, 0)
		exportedTargets     = map[string]string{}

		blueprintDataImports       = sets.NewString()
		blueprintTargetImports     = sets.NewString()
		blueprintTargetListImports = sets.NewString()
	)

	for _, bImport := range blueprintImportDefs {
		if len(bImport.Type) != 0 {
			switch bImport.Type {
			case core.ImportTypeData:
				blueprintDataImports.Insert(bImport.Name)
			case core.ImportTypeTarget:
				blueprintTargetImports.Insert(bImport.Name)
			case core.ImportTypeTargetList:
				blueprintTargetListImports.Insert(bImport.Name)
			}
		} else {
			// fallback to old logic
			if bImport.Schema != nil {
				blueprintDataImports.Insert(bImport.Name)
			} else if len(bImport.TargetType) != 0 {
				blueprintTargetImports.Insert(bImport.Name)
			}
		}

	}

	for i, instTmpl := range subinstallations {
		instPath := fldPath.Index(i)
		if len(instTmpl.Name) != 0 {
			instPath = fldPath.Child(instTmpl.Name)
		}

		//if len(subinst.File) != 0 {
		//	data, err := vfs.ReadFile(fs, subinst.File)
		//	if err != nil {
		//		if os.IsNotExist(err) {
		//			allErrs = append(allErrs, field.NotFound(instPath.Child("file"), subinst.File))
		//			continue
		//		}
		//		allErrs = append(allErrs, field.InternalError(instPath.Child("file"), err))
		//		continue
		//	}
		//
		//	instTmpl = &core.InstallationTemplate{}
		//
		//	if _, _, err := serializer.NewCodecFactory(landscaperScheme).UniversalDecoder().Decode(data, nil, instTmpl); err != nil {
		//		allErrs = append(allErrs, field.Invalid(instPath.Child("file"), subinst.File, err.Error()))
		//		continue
		//	}
		//}

		for i, do := range instTmpl.Exports.Data {
			dataPath := instPath.Child("exports").Child("data").Index(i).Key(fmt.Sprintf("%s/%s", do.Name, do.DataRef))
			if dup, ok := exportedDataObjects[do.DataRef]; ok {
				allErrs = append(allErrs, field.Forbidden(dataPath, fmt.Sprintf("data export '%s' is already exported by %s", do.DataRef, dup)))
			} else {
				exportedDataObjects[do.DataRef] = dataPath.String()
			}
			if blueprintDataImports.Has(do.DataRef) {
				allErrs = append(allErrs, field.Forbidden(dataPath, "export is imported by its parent"))
			}
		}
		for i, do := range instTmpl.Imports.Data {
			importedDataObjects = append(importedDataObjects, Import{
				Name: do.DataRef,
				Path: instPath.Child("imports").Child("data").Index(i).Key(do.Name),
			})
		}
		for i, target := range instTmpl.Exports.Targets {
			targetPath := instPath.Child("exports").Child("targets").Index(i).Key(fmt.Sprintf("%s/%s", target.Name, target.Target))
			if dup, ok := exportedTargets[target.Target]; ok {
				allErrs = append(allErrs, field.Forbidden(targetPath, fmt.Sprintf("target export '%s' is already exported by %s", target.Target, dup)))
			} else {
				exportedTargets[target.Target] = targetPath.String()
			}
			if blueprintTargetImports.Has(target.Target) || blueprintTargetListImports.Has(target.Target) {
				allErrs = append(allErrs, field.Forbidden(targetPath, "export is imported by its parent"))
			}
		}
		for i, target := range instTmpl.Imports.Targets {
			impPath := instPath.Child("imports").Child("targets").Index(i).Key(target.Name)
			if len(target.Target) != 0 {
				importedTargets = append(importedTargets, Import{
					Name: target.Target,
					Path: impPath,
				})
			} else if target.Targets != nil {
				for i2, t2 := range target.Targets {
					importedTargets = append(importedTargets, Import{
						Name: t2,
						Path: impPath.Child("targets").Index(i2),
					})
				}
			} else if len(target.TargetListReference) != 0 {
				importedTargets = append(importedTargets, Import{
					Name:         target.TargetListReference,
					Path:         impPath,
					IsListImport: true,
				})
			}
			// invalid definition if no if matches, but this is validated at another point already
		}

		allErrs = append(allErrs, ValidateInstallationTemplate(instPath, instTmpl)...)
		if len(instTmpl.Name) != 0 && names.Has(instTmpl.Name) {
			allErrs = append(allErrs, field.Duplicate(instPath, "duplicated subinstallation"))
		}
		names.Insert(instTmpl.Name)
	}

	// validate that all imported values are either satisfied by the blueprint or by another sibling
	allErrs = append(allErrs, ValidateSatisfiedImports(blueprintDataImports, nil, sets.StringKeySet(exportedDataObjects), importedDataObjects)...)
	allErrs = append(allErrs, ValidateSatisfiedImports(blueprintTargetImports, blueprintTargetListImports, sets.StringKeySet(exportedTargets), importedTargets)...)

	return allErrs
}

// Import defines a internal import struct for validation.
type Import struct {
	Name         string
	Path         *field.Path
	IsListImport bool
}

// ValidateSatisfiedImports validates that all imports are satisfied.
func ValidateSatisfiedImports(blueprintImports, blueprintListImports, exports sets.String, imports []Import) field.ErrorList { //nolint:staticcheck // Ignore SA1019 // TODO: change to generic set
	allErrs := field.ErrorList{}
	for _, imp := range imports {
		if len(blueprintListImports) > 0 { // no need to check for references to elements from targetlist imports, if there aren't any targetlist imports
			if imp.IsListImport {
				if !blueprintListImports.Has(imp.Name) {
					allErrs = append(allErrs, field.NotFound(imp.Path, "import not satisfied"))
				}
				continue
			}
			isIndexed, impName, _, err := IsIndexed(imp.Name)
			if err != nil {
				allErrs = append(allErrs, field.Invalid(imp.Path, imp.Name, err.Error()))
				continue
			}
			if isIndexed {
				if !blueprintListImports.Has(impName) {
					allErrs = append(allErrs, field.NotFound(imp.Path, "import not satisfied"))
				}
				continue
			}
		}
		if imp.IsListImport || !exports.Has(imp.Name) && !blueprintImports.Has(imp.Name) {
			allErrs = append(allErrs, field.NotFound(imp.Path, "import not satisfied"))
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

	if len(template.Blueprint.Ref) == 0 && len(template.Blueprint.Filesystem.RawMessage) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("blueprint"), "a blueprint must be defined"))
	}

	allErrs = append(allErrs, ValidateInstallationTemplateImports(template.Imports, fldPath.Child("imports"))...)
	allErrs = append(allErrs, ValidateInstallationExports(template.Exports, fldPath.Child("exports"))...)

	return allErrs
}

// ValidateInstallationTemplateImports validates the imports of an InstallationTemplate
func ValidateInstallationTemplateImports(imports core.InstallationImports, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	importNames := sets.NewString()
	var tmpErrs field.ErrorList

	tmpErrs, importNames = ValidateInstallationTemplateDataImports(imports.Data, fldPath.Child("data"), importNames)
	allErrs = append(allErrs, tmpErrs...)
	tmpErrs, _ = ValidateInstallationTargetImports(imports.Targets, fldPath.Child("targets"), importNames)
	allErrs = append(allErrs, tmpErrs...)

	return allErrs
}

// ValidateInstallationTemplateDataImports validates the data imports of an InstallationTemplate
func ValidateInstallationTemplateDataImports(imports []core.DataImport, fldPath *field.Path, importNames sets.String) (field.ErrorList, sets.String) { //nolint:staticcheck // Ignore SA1019 // TODO: change to generic set
	allErrs := field.ErrorList{}

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
		if importNames.Has(imp.Name) {
			allErrs = append(allErrs, field.Duplicate(fldPath.Index(idx), imp.Name))
		}
		importNames.Insert(imp.Name)
	}

	return allErrs, importNames
}
