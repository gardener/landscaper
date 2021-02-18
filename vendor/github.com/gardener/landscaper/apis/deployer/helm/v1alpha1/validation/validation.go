// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
)

// ValidateProviderConfiguration validates a helm deployer configuration
func ValidateProviderConfiguration(config *helmv1alpha1.ProviderConfiguration) error {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateTimeout(field.NewPath("deleteTimeout"), config.DeleteTimeout)...)
	allErrs = append(allErrs, ValidateTimeout(field.NewPath("healthChecks", "timeout"), config.HealthChecks.Timeout)...)
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
	if len(chart.Ref) == 0 && chart.Archive == nil && chart.FromResource == nil {
		return append(allErrs, field.Required(fldPath.Child("ref", "archive", "fromResource"), "must not be empty"))
	}

	if chart.Archive != nil {
		allErrs = append(allErrs, ValidateArchive(fldPath.Child("archive"), chart.Archive)...)
	} else if chart.FromResource != nil {
		allErrs = append(allErrs, ValidateFromResource(fldPath.Child("fromResource"), chart.FromResource)...)
	}

	return allErrs
}

// ValidateArchive validates the archive access for a helm chart.
func ValidateArchive(fldPath *field.Path, archive *helmv1alpha1.ArchiveAccess) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(archive.Raw) == 0 && archive.Remote == nil {
		return append(allErrs, field.Required(fldPath.Child("raw", "remote"), "must not be empty"))
	}

	if archive.Remote != nil {
		remotePath := fldPath.Child("remote")
		if len(archive.Remote.URL) == 0 {
			allErrs = append(allErrs, field.Required(remotePath.Child("url"), "must not be empty"))
		}
	}

	return allErrs
}

// ValidateFromResource validates the resource access for a helm chart.
func ValidateFromResource(fldPath *field.Path, resourceRef *helmv1alpha1.RemoteChartReference) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(resourceRef.ResourceName) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("resourceName"), "must not be empty"))
	}

	if resourceRef.Inline != nil {
		return allErrs
	}

	if resourceRef.Reference == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("ref"), "must not be empty"))
	} else {
		if resourceRef.Reference.RepositoryContext == nil {
			allErrs = append(allErrs, field.Required(fldPath.Child("repositoryContext"), "must not be empty"))
		}

		if len(resourceRef.Reference.ComponentName) == 0 {
			allErrs = append(allErrs, field.Required(fldPath.Child("componentName"), "must not be empty"))
		}
		if len(resourceRef.Reference.Version) == 0 {
			allErrs = append(allErrs, field.Required(fldPath.Child("version"), "must not be empty"))
		}
	}

	return allErrs
}

// ValidateTimeout validates that a timeout can be parsed as Duration.
func ValidateTimeout(fldPath *field.Path, timeout string) field.ErrorList {
	allErrs := field.ErrorList{}
	if strings.HasPrefix(timeout, "-") {
		allErrs = append(allErrs, field.Invalid(fldPath, timeout, "timeout can not be negative"))
	}
	if _, err := time.ParseDuration(timeout); err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath, timeout, "invalid duration string"))
	}
	return allErrs
}
