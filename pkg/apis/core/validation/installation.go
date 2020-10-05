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
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/pkg/apis/core"
)

// ValidateInstallation validates an Installation
func ValidateInstallation(inst *core.Installation) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, apivalidation.ValidateObjectMeta(&inst.ObjectMeta, true, apivalidation.NameIsDNSLabel, field.NewPath("metadata"))...)

	return allErrs
}

// ValidateInstallationSpec validates the spec of an Installation
func ValidateInstallationSpec(spec *core.InstallationSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, ValidateInstallationImports(spec.Imports, fldPath.Child("imports"))...)
	allErrs = append(allErrs, ValidateInstallationExports(spec.Exports, fldPath.Child("exports"))...)

	// check that either inline blueprint or reference is provided (and not both)
	if spec.Blueprint.Reference == nil && spec.Blueprint.Inline == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("blueprint"), "must specify either inline blueprint or reference"))
	}
	if spec.Blueprint.Reference != nil && spec.Blueprint.Inline != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("blueprint"), spec.Blueprint, "must specify either inline blueprint or reference, not both"))
	}

	// check RegistryPullSecrets
	allErrs = append(allErrs, ValidateObjectReferenceList(spec.RegistryPullSecrets, fldPath.Child("registryPullSecrets"))...)

	return allErrs
}

// ValidateInstallationImports validates the imports of an Installation
func ValidateInstallationImports(imports *core.InstallationImports, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, ValidateInstallationDataImports(imports.Data, fldPath.Child("data"))...)
	allErrs = append(allErrs, ValidateInstallationTargetImports(imports.Targets, fldPath.Child("targets"))...)

	return allErrs
}

// ValidateInstallationDataImports validates the data imports of an Installation
func ValidateInstallationDataImports(imports []core.DataImport, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	importNames := map[string]bool{}
	for idx, imp := range imports {
		if imp.DataRef == "" {
			allErrs = append(allErrs, field.Required(fldPath.Index(idx).Child("dataRef"), "dataRef must not be empty"))
		}
		if imp.Name == "" {
			allErrs = append(allErrs, field.Required(fldPath.Index(idx).Child("name"), "name must not be empty"))
			continue
		}
		if importNames[imp.Name] {
			allErrs = append(allErrs, field.Duplicate(fldPath.Index(idx), imp.Name))
		}
		importNames[imp.Name] = true
	}

	return allErrs
}

// ValidateInstallationTargetImports validates the target imports of an Installation
func ValidateInstallationTargetImports(imports []core.TargetImportExport, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	importNames := map[string]bool{}
	for idx, imp := range imports {
		if imp.Target == "" {
			allErrs = append(allErrs, field.Required(fldPath.Index(idx).Child("target"), "target must not be empty"))
		}
		if imp.Name == "" {
			allErrs = append(allErrs, field.Required(fldPath.Index(idx).Child("name"), "name must not be empty"))
			continue
		}
		if importNames[imp.Name] {
			allErrs = append(allErrs, field.Duplicate(fldPath.Index(idx), imp.Name))
		}
		importNames[imp.Name] = true
	}

	return allErrs
}

// ValidateInstallationExports validates the exports of an Installation
func ValidateInstallationExports(exports *core.InstallationExports, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, ValidateInstallationDataExports(exports.Data, fldPath.Child("data"))...)
	allErrs = append(allErrs, ValidateInstallationTargetExports(exports.Targets, fldPath.Child("targets"))...)

	return allErrs
}

// ValidateInstallationDataExports validates the data exports of an Installation
func ValidateInstallationDataExports(exports []core.DataExport, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	importNames := map[string]bool{}
	for idx, imp := range exports {
		if imp.DataRef == "" {
			allErrs = append(allErrs, field.Required(fldPath.Index(idx).Child("dataRef"), "dataRef must not be empty"))
		}
		if imp.Name == "" {
			allErrs = append(allErrs, field.Required(fldPath.Index(idx).Child("name"), "name must not be empty"))
			continue
		}
		if importNames[imp.Name] {
			allErrs = append(allErrs, field.Duplicate(fldPath.Index(idx), imp.Name))
		}
		importNames[imp.Name] = true
	}

	return allErrs
}

// ValidateInstallationTargetExports validates the target exports of an Installation
func ValidateInstallationTargetExports(exports []core.TargetImportExport, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	importNames := map[string]bool{}
	for idx, imp := range exports {
		if imp.Target == "" {
			allErrs = append(allErrs, field.Required(fldPath.Index(idx).Child("target"), "target must not be empty"))
		}
		if imp.Name == "" {
			allErrs = append(allErrs, field.Required(fldPath.Index(idx).Child("name"), "name must not be empty"))
			continue
		}
		if importNames[imp.Name] {
			allErrs = append(allErrs, field.Duplicate(fldPath.Index(idx), imp.Name))
		}
		importNames[imp.Name] = true
	}

	return allErrs
}

// ValidateObjectReference validates that the object reference is valid
func ValidateObjectReference(or core.ObjectReference, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if or.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "name must not be empty"))
	}
	if or.Namespace == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("namespace"), "namespace must not be empty"))
	}

	return allErrs
}

// ValidateObjectReferenceList validates a list of object references
func ValidateObjectReferenceList(orl []core.ObjectReference, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for i, e := range orl {
		allErrs = append(allErrs, ValidateObjectReference(e, fldPath.Index(i))...)
	}

	return allErrs
}
