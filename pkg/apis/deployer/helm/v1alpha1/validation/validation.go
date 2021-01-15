// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	helmv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/helm/v1alpha1"
)

// ValidateProviderConfiguration validates a helm deployer configuration
func ValidateProviderConfiguration(config *helmv1alpha1.ProviderConfiguration) error {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateChart(field.NewPath("chart"), config.Chart)...)

	expPath := field.NewPath("exportsFromManifests")
	keys := sets.NewString()
	for i, export := range config.ExportsFromManifests {
		indexFldPath := expPath.Index(i)
		if len(export.Key) == 0 {
			allErrs = append(allErrs, field.Required(indexFldPath.Child("key"), "must not be empty"))
		}

		if keys.Has(export.Key) {
			allErrs = append(allErrs, field.Duplicate(indexFldPath.Child("key"), fmt.Sprintf("duplicated key %s is not allowed", export.Key)))
		}
		keys.Insert(export.Key)

		if len(export.JSONPath) == 0 {
			allErrs = append(allErrs, field.Required(indexFldPath.Child("jsonPath"), "must not be empty"))
		}

		if export.FromResource != nil {
			resFldPath := indexFldPath.Child("resource")
			if len(export.FromResource.APIVersion) == 0 {
				allErrs = append(allErrs, field.Required(resFldPath.Child("apiGroup"), "must not be empty"))
			}
			if len(export.FromResource.Kind) == 0 {
				allErrs = append(allErrs, field.Required(resFldPath.Child("kind"), "must not be empty"))
			}
			if len(export.FromResource.Name) == 0 {
				allErrs = append(allErrs, field.Required(resFldPath.Child("name"), "must not be empty"))
			}
			if len(export.FromResource.Namespace) == 0 {
				allErrs = append(allErrs, field.Required(resFldPath.Child("namespace"), "must not be empty"))
			}
		}
	}

	return allErrs.ToAggregate()
}

// ValidateChart validates the access methods for a chart
func ValidateChart(fldPath *field.Path, chart helmv1alpha1.Chart) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(chart.Ref) == 0 && len(chart.Tar) == 0 {
		return append(allErrs, field.Required(fldPath.Child("ref", "tar"), "must not be empty"))
	}
	return allErrs
}
