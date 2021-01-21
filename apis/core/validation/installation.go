// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/apis/core/v1alpha1/helper"

	"github.com/gardener/landscaper/apis/core"
)

// InstallationNameMaxLength is the max allowed length of an installation name
const InstallationNameMaxLength = validation.DNS1123LabelMaxLength - len(helper.InstallationPrefix)

// InstallationGenerateNameMaxLength is the max length of an installation name minus the number of random characters kubernetes uses to generate a unique name
const InstallationGenerateNameMaxLength = InstallationNameMaxLength - 5

// ValidateInstallation validates an Installation
func ValidateInstallation(inst *core.Installation) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateInstallationObjectMeta(&inst.ObjectMeta, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateInstallationSpec(&inst.Spec, field.NewPath("spec"))...)
	return allErrs
}

func validateInstallationObjectMeta(objMeta *metav1.ObjectMeta, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, apivalidation.ValidateObjectMeta(objMeta, true, apivalidation.NameIsDNSLabel, fldPath)...)

	if len(objMeta.GetName()) > InstallationNameMaxLength {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("name"), objMeta.GetName(), validation.MaxLenError(InstallationNameMaxLength)))
	} else if len(objMeta.GetGenerateName()) > InstallationGenerateNameMaxLength {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("generateName"), objMeta.GetGenerateName(), validation.MaxLenError(InstallationGenerateNameMaxLength)))
	}

	return allErrs
}

// ValidateInstallationSpec validates the spec of an Installation
func ValidateInstallationSpec(spec *core.InstallationSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, ValidateInstallationImports(spec.Imports, fldPath.Child("imports"))...)
	allErrs = append(allErrs, ValidateInstallationExports(spec.Exports, fldPath.Child("exports"))...)

	// check Blueprint and ComponentDescriptor
	allErrs = append(allErrs, ValidateInstallationBlueprint(spec.Blueprint, fldPath.Child("blueprint"))...)
	allErrs = append(allErrs, ValidateInstallationComponentDescriptor(spec.ComponentDescriptor, fldPath.Child("componentDescriptor"))...)

	// check RegistryPullSecrets
	allErrs = append(allErrs, ValidateObjectReferenceList(spec.RegistryPullSecrets, fldPath.Child("registryPullSecrets"))...)

	return allErrs
}

// ValidateInstallationBlueprint validates the Blueprint definition of an Installation
func ValidateInstallationBlueprint(bp core.BlueprintDefinition, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// check that either inline blueprint or reference is provided (and not both)
	if bp.Reference == nil && bp.Inline == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("definition"), "must specify either inline blueprint or reference"))
	}
	if bp.Reference != nil && bp.Inline != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("definition"), bp, "must specify either inline blueprint or reference, not both"))
	}

	return allErrs
}

// ValidateInstallationComponentDescriptor validates the ComponentDesriptor of an Installation
func ValidateInstallationComponentDescriptor(cd *core.ComponentDescriptorDefinition, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// check that a ComponentDescriptor - if given - is either inline or ref but not both
	if cd != nil {
		if cd.Inline == nil && cd.Reference == nil {
			allErrs = append(allErrs, field.Required(fldPath.Child("definition"), "must specify either inline Component Descriptor or reference if a Component Descriptor is supplied"))
		}
		if cd.Inline != nil && cd.Reference != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("definition"), *cd, "must specify either inline Component Descriptor or reference - not both - if a Component Descriptor is supplied "))
		}
	}

	return allErrs
}

// ValidateInstallationImports validates the imports of an Installation
func ValidateInstallationImports(imports core.InstallationImports, fldPath *field.Path) field.ErrorList {
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
		impPath := fldPath.Index(idx)

		if imp.DataRef == "" && imp.SecretRef == nil && imp.ConfigMapRef == nil {
			allErrs = append(allErrs, field.Required(impPath, "either dataRef, secretRef or configMapRef must not be empty"))
		}

		if imp.SecretRef != nil {
			secRefField := impPath.Child("secretRef")
			allErrs = append(allErrs, ValidateSecretReference(*imp.SecretRef, secRefField)...)
			if imp.DataRef != "" || imp.ConfigMapRef != nil {
				allErrs = append(allErrs, field.Forbidden(secRefField, "multiple data references are defined"))
			}
		}

		if imp.ConfigMapRef != nil {
			cmRefField := impPath.Child("configMapRef")
			allErrs = append(allErrs, ValidateConfigMapReference(*imp.ConfigMapRef, cmRefField)...)
			if imp.DataRef != "" || imp.SecretRef != nil {
				allErrs = append(allErrs, field.Forbidden(cmRefField, "multiple data references are defined"))
			}
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
		}
		if importNames[imp.Name] {
			allErrs = append(allErrs, field.Duplicate(fldPath.Index(idx), imp.Name))
		}
		importNames[imp.Name] = true
	}

	return allErrs
}

// ValidateInstallationExports validates the exports of an Installation
func ValidateInstallationExports(exports core.InstallationExports, fldPath *field.Path) field.ErrorList {
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

// ValidateSecretReference validates that the secret reference is valid
func ValidateSecretReference(sr core.SecretReference, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, ValidateObjectReference(sr.ObjectReference, fldPath)...)
	if sr.Key == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("key"), "key must not be empty"))
	}

	return allErrs
}

// ValidateConfigMapReference validates that the secret reference is valid
func ValidateConfigMapReference(cmr core.ConfigMapReference, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, ValidateObjectReference(cmr.ObjectReference, fldPath)...)
	if cmr.Key == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("key"), "key must not be empty"))
	}

	return allErrs
}
