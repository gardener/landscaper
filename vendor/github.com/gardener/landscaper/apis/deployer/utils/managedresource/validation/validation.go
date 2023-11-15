// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
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
	if len(ref.APIVersion) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("apiVersion"), "must not be empty"))
	}
	if len(ref.Kind) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("kind"), "must not be empty"))
	}
	if len(ref.JSONPath) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("jsonPath"), "must not be empty"))
	}
	return allErrs
}

func ValidateDeletionGroups(fldPath *field.Path, groups []managedresource.DeletionGroupDefinition) field.ErrorList {
	var allErrs field.ErrorList
	for i, g := range groups {
		allErrs = append(allErrs, validateDeletionGroup(fldPath.Index(i), &g)...)
	}
	return allErrs
}

func validateDeletionGroup(fldPath *field.Path, g *managedresource.DeletionGroupDefinition) field.ErrorList {
	var allErrs field.ErrorList

	if g.IsPredefined() && g.IsCustom() {
		allErrs = append(allErrs, field.Invalid(fldPath, g, "predefinedResourceGroup and customResourceGroup must not both be set"))
	}
	if !g.IsPredefined() && !g.IsCustom() {
		allErrs = append(allErrs, field.Invalid(fldPath, g, "either predefinedResourceGroup or customResourceGroup must be set"))
	}
	if g.IsPredefined() {
		allErrs = append(allErrs, validatePredefinedResourceGroup(fldPath.Child("predefinedResourceGroup"), g.PredefinedResourceGroup)...)
	}
	if g.IsCustom() {
		allErrs = append(allErrs, validateCustomResourceGroup(fldPath.Child("customResourceGroup"), g.CustomResourceGroup)...)
	}

	return allErrs
}

func validatePredefinedResourceGroup(fldPath *field.Path, p *managedresource.PredefinedResourceGroup) field.ErrorList {
	var allErrs field.ErrorList

	switch p.Type {
	case managedresource.PredefinedResourceGroupNamespacedResources,
		managedresource.PredefinedResourceGroupClusterScopedResources,
		managedresource.PredefinedResourceGroupCRDs,
		managedresource.PredefinedResourceGroupEmpty:
	case "":
		allErrs = append(allErrs, field.Required(fldPath.Child("type"), "must not be empty"))
	default:
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("type"), p.Type, []string{
			string(managedresource.PredefinedResourceGroupNamespacedResources),
			string(managedresource.PredefinedResourceGroupClusterScopedResources),
			string(managedresource.PredefinedResourceGroupCRDs),
			string(managedresource.PredefinedResourceGroupEmpty),
		}))
	}

	return allErrs
}

func validateCustomResourceGroup(fldPath *field.Path, c *managedresource.CustomResourceGroup) field.ErrorList {
	var allErrs field.ErrorList

	if len(c.Resources) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("resources"), "must not be empty"))
	}

	return allErrs
}
