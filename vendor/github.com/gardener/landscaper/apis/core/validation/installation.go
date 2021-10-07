// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
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
	allErrs = append(allErrs, ValidateExactlyOneOf(fldPath.Child("definition"), bp, "Inline", "Reference")...)

	return allErrs
}

// ValidateInstallationComponentDescriptor validates the ComponentDesriptor of an Installation
func ValidateInstallationComponentDescriptor(cd *core.ComponentDescriptorDefinition, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// check that a ComponentDescriptor - if given - is either inline or ref but not both
	if cd != nil {
		allErrs = append(allErrs, ValidateExactlyOneOf(fldPath.Child("definition"), *cd, "Inline", "Reference")...)
	}

	return allErrs
}

// ValidateInstallationImports validates the imports of an Installation
func ValidateInstallationImports(imports core.InstallationImports, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	importNames := sets.NewString()
	var tmpErrs field.ErrorList

	tmpErrs, importNames = ValidateInstallationDataImports(imports.Data, fldPath.Child("data"), importNames)
	allErrs = append(allErrs, tmpErrs...)
	tmpErrs, importNames = ValidateInstallationTargetImports(imports.Targets, fldPath.Child("targets"), importNames)
	allErrs = append(allErrs, tmpErrs...)
	tmpErrs, _ = ValidateInstallationComponentDescriptorImports(imports.ComponentDescriptors, fldPath.Child("componentDescriptors"), importNames)
	allErrs = append(allErrs, tmpErrs...)

	return allErrs
}

// ValidateInstallationDataImports validates the data imports of an Installation
func ValidateInstallationDataImports(imports []core.DataImport, fldPath *field.Path, importNames sets.String) (field.ErrorList, sets.String) {
	allErrs := field.ErrorList{}

	for idx, imp := range imports {
		impPath := fldPath.Index(idx)

		allErrs = append(allErrs, ValidateExactlyOneOf(impPath, imp, "DataRef", "SecretRef", "ConfigMapRef")...)

		if imp.SecretRef != nil {
			allErrs = append(allErrs, ValidateSecretReference(*imp.SecretRef, impPath.Child("secretRef"))...)
		}

		if imp.ConfigMapRef != nil {
			allErrs = append(allErrs, ValidateConfigMapReference(*imp.ConfigMapRef, impPath.Child("configMapRef"))...)
		}

		if imp.Name == "" {
			allErrs = append(allErrs, field.Required(impPath.Child("name"), "name must not be empty"))
			continue
		}
		if importNames.Has(imp.Name) {
			allErrs = append(allErrs, field.Duplicate(impPath, imp.Name))
		}
		importNames.Insert(imp.Name)
	}

	return allErrs, importNames
}

// ValidateInstallationTargetImports validates the target imports of an Installation
func ValidateInstallationTargetImports(imports []core.TargetImport, fldPath *field.Path, importNames sets.String) (field.ErrorList, sets.String) {
	allErrs := field.ErrorList{}

	for idx, imp := range imports {
		fldPathIdx := fldPath.Index(idx)
		if imp.Name == "" {
			allErrs = append(allErrs, field.Required(fldPathIdx.Child("name"), "name must not be empty"))
		}
		allErrs = append(allErrs, ValidateExactlyOneOf(fldPathIdx, imp, "Target", "Targets", "TargetListReference")...)
		if len(imp.Targets) > 0 {
			for idx2, tg := range imp.Targets {
				if len(tg) == 0 {
					allErrs = append(allErrs, field.Required(fldPathIdx.Child("targets").Index(idx2), "target must not be empty"))
				}
			}
		}
		if importNames.Has(imp.Name) {
			allErrs = append(allErrs, field.Duplicate(fldPathIdx, imp.Name))
		}
		importNames.Insert(imp.Name)
	}

	return allErrs, importNames
}

// ValidateInstallationComponentDescriptorImports validates the component descriptor imports of an Installation
func ValidateInstallationComponentDescriptorImports(imports []core.ComponentDescriptorImport, fldPath *field.Path, importNames sets.String) (field.ErrorList, sets.String) {
	allErrs := field.ErrorList{}

	for idx, imp := range imports {
		fldPathIdx := fldPath.Index(idx)
		if imp.Name == "" {
			allErrs = append(allErrs, field.Required(fldPathIdx.Child("name"), "name must not be empty"))
		}
		allErrs = append(allErrs, ValidateExactlyOneOf(fldPathIdx, imp, "Ref", "ConfigMapRef", "SecretRef", "List")...)
		if imp.ConfigMapRef != nil {
			allErrs = append(allErrs, ValidateConfigMapReference(*imp.ConfigMapRef, fldPathIdx.Child("configMapRef"))...)
		}
		if imp.SecretRef != nil {
			allErrs = append(allErrs, ValidateSecretReference(*imp.SecretRef, fldPathIdx.Child("secretRef"))...)
		}
		if len(imp.DataRef) != 0 {
			allErrs = append(allErrs, field.Invalid(fldPathIdx.Child("dataRef"), imp.DataRef, "must be set in subinstallation templates only"))
		}
		if len(imp.List) > 0 {
			for idx2, cd := range imp.List {
				fldPathIdx2 := fldPathIdx.Child("list").Index(idx2)
				allErrs = append(allErrs, ValidateExactlyOneOf(fldPathIdx2, cd, "Ref", "ConfigMapRef", "SecretRef")...)
				if cd.ConfigMapRef != nil {
					allErrs = append(allErrs, ValidateConfigMapReference(*cd.ConfigMapRef, fldPathIdx2.Child("configMapRef"))...)
				}
				if cd.SecretRef != nil {
					allErrs = append(allErrs, ValidateSecretReference(*cd.SecretRef, fldPathIdx2.Child("secretRef"))...)
				}
				if len(cd.DataRef) != 0 {
					allErrs = append(allErrs, field.Invalid(fldPathIdx2.Child("dataRef"), cd.DataRef, "must be set in subinstallation templates only"))
				}
			}
		}
		if importNames.Has(imp.Name) {
			allErrs = append(allErrs, field.Duplicate(fldPathIdx, imp.Name))
		}
		importNames.Insert(imp.Name)
	}

	return allErrs, importNames
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
func ValidateInstallationTargetExports(exports []core.TargetExport, fldPath *field.Path) field.ErrorList {
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
	return allErrs
}

// ValidateConfigMapReference validates that the secret reference is valid
func ValidateConfigMapReference(cmr core.ConfigMapReference, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateObjectReference(cmr.ObjectReference, fldPath)...)
	return allErrs
}
