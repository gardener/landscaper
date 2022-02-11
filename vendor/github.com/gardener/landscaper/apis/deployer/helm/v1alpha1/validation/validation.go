// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	crval "github.com/gardener/landscaper/apis/deployer/utils/continuousreconcile/validation"
	health "github.com/gardener/landscaper/apis/deployer/utils/readinesschecks/validation"
)

const (
	helmArgumentAtomic  = "atomic"
	helmArgumentTimeout = "timeout"
)

// ValidateProviderConfiguration validates a helm deployer configuration
func ValidateProviderConfiguration(config *helmv1alpha1.ProviderConfiguration) error {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateTimeout(field.NewPath("deleteTimeout"), config.DeleteTimeout)...)
	allErrs = append(allErrs, ValidateTimeout(field.NewPath("readinessChecks", "timeout"), config.ReadinessChecks.Timeout)...)
	allErrs = append(allErrs, health.ValidateReadinessCheckConfiguration(field.NewPath("readinessChecks"), &config.ReadinessChecks)...)
	allErrs = append(allErrs, ValidateChart(field.NewPath("chart"), config.Chart)...)
	allErrs = append(allErrs, ValidateHelmDeploymentConfiguration(field.NewPath("helmDeploymentConfig"), config.HelmDeploymentConfig)...)
	allErrs = append(allErrs, crval.ValidateContinuousReconcileSpec(field.NewPath("continuousReconcile"), config.ContinuousReconcile)...)

	if len(config.Name) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("name"), "must not be empty"))
	}
	if len(config.Namespace) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("namespace"), "must not be empty"))
	}

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
	if len(chart.Ref) == 0 && chart.Archive == nil && chart.FromResource == nil && chart.HelmChartRepo == nil {
		subPath := fldPath.Child("ref", "archive", "fromResource", "helmChartRepo")
		err := field.Required(subPath, "must not be empty")
		return append(allErrs, err)
	}

	if chart.Archive != nil {
		allErrs = append(allErrs, ValidateArchive(fldPath.Child("archive"), chart.Archive)...)
	} else if chart.FromResource != nil {
		allErrs = append(allErrs, ValidateFromResource(fldPath.Child("fromResource"), chart.FromResource)...)
	} else if chart.HelmChartRepo != nil {
		allErrs = append(allErrs, ValidateHelmChartRepo(fldPath.Child("helmChartRepo"), chart.HelmChartRepo)...)
	}

	return allErrs
}

func ValidateHelmDeploymentConfiguration(fldPath *field.Path, deployConfig *helmv1alpha1.HelmDeploymentConfiguration) field.ErrorList {
	allErrs := field.ErrorList{}
	if deployConfig != nil {
		allErrs = append(allErrs, ValidateInstallConfiguration(fldPath.Child("install"), deployConfig.Install)...)
		allErrs = append(allErrs, ValidateUpgradeConfiguration(fldPath.Child("upgrade"), deployConfig.Upgrade)...)
		allErrs = append(allErrs, ValidateUninstallConfiguration(fldPath.Child("uninstall"), deployConfig.Uninstall)...)
	}
	return allErrs
}

func ValidateInstallConfiguration(fldPath *field.Path, conf map[string]lsv1alpha1.AnyJSON) field.ErrorList {
	return validateHelmArguments(fldPath, conf, []string{helmArgumentAtomic, helmArgumentTimeout})
}

func ValidateUpgradeConfiguration(fldPath *field.Path, conf map[string]lsv1alpha1.AnyJSON) field.ErrorList {
	return validateHelmArguments(fldPath, conf, []string{helmArgumentAtomic, helmArgumentTimeout})
}

func ValidateUninstallConfiguration(fldPath *field.Path, conf map[string]lsv1alpha1.AnyJSON) field.ErrorList {
	return validateHelmArguments(fldPath, conf, []string{helmArgumentTimeout})
}

func validateHelmArguments(fldPath *field.Path, conf map[string]lsv1alpha1.AnyJSON, validArguments []string) field.ErrorList {
	allErrs := field.ErrorList{}

	for key := range conf {
		found := false
		for _, validArg := range validArguments {
			if key == validArg {
				found = true
				break
			}
		}
		if !found {
			err := field.NotSupported(fldPath, key, validArguments)
			allErrs = append(allErrs, err)
		}
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

// ValidateHelmChartRepo validates the helm chart repo access for a helm chart.
func ValidateHelmChartRepo(fldPath *field.Path, helmChartRepo *helmv1alpha1.HelmChartRepo) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(helmChartRepo.HelmChartRepoUrl) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("helmChartRepoUrl"), "must not be empty"))
	}

	if len(helmChartRepo.HelmChartName) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("helmChartName"), "must not be empty"))
	}

	if len(helmChartRepo.HelmChartVersion) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("helmChartVersion"), "must not be empty"))
	}

	return allErrs
}

// ValidateTimeout validates that a timeout can be parsed as Duration.
func ValidateTimeout(fldPath *field.Path, timeout *lsv1alpha1.Duration) field.ErrorList {
	allErrs := field.ErrorList{}
	if timeout == nil {
		allErrs = append(allErrs, field.Required(fldPath, "timeout can not be empty"))
		return allErrs
	}
	if timeout.Duration < 0 {
		allErrs = append(allErrs, field.Invalid(fldPath, timeout, "timeout can not be negative"))
	}
	return allErrs
}
