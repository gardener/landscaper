// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// ValidateManifestList validates a list of manifests.
func ValidateManifestList(fldPath *field.Path, list []managedresource.Manifest) field.ErrorList {
	var allErrs field.ErrorList
	for i, m := range list {
		allErrs = append(allErrs, ValidateManifest(fldPath.Index(i), m)...)
	}
	return allErrs
}

// ValidateManifest validates a manifest.
func ValidateManifest(fldPath *field.Path, manifest managedresource.Manifest) field.ErrorList {
	var allErrs field.ErrorList
	if manifest.Manifest == nil || len(manifest.Manifest.Raw) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("manifest"), "manifest must be defined"))
	}
	return allErrs
}

// ValidateManifestExport validates a readiness check configuration
func ValidateManifestExport(fldPath *field.Path, export *managedresource.Export) field.ErrorList {
	var allErrs field.ErrorList

	if len(export.Key) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("key"), "must not be empty"))
	}
	if len(export.JSONPath) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("jsonPath"), "must not be empty"))
	}
	if export.FromResource != nil {
		allErrs = append(allErrs, ValidateTypedObjectReference(fldPath.Child("fromResource"), export.FromResource)...)
	}
	if export.FromObjectReference != nil {
		allErrs = append(allErrs, ValidateFromObjectReference(fldPath.Child("fromObjectRef"), export.FromObjectReference)...)
	}

	return allErrs
}

// ValidateTypedObjectReference validates a typed object reference.
func ValidateTypedObjectReference(fldPath *field.Path, ref *lsv1alpha1.TypedObjectReference) field.ErrorList {
	var allErrs field.ErrorList

	if len(ref.APIVersion) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("apiVersion"), "must not be empty"))
	}
	if len(ref.Kind) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("kind"), "must not be empty"))
	}
	if len(ref.Name) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "must not be empty"))
	}
	// namespace can be empty as we could reference a clusterwide resource

	return allErrs
}

// ValidateFromObjectReference validates a typed object reference.
func ValidateFromObjectReference(fldPath *field.Path, ref *managedresource.FromObjectReference) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, ValidateTypedObjectReference(fldPath, &ref.TypedObjectReference)...)
	if len(ref.JSONPath) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("jsonPath"), "must not be empty"))
	}
	return allErrs
}
