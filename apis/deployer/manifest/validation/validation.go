// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	healthchecks "github.com/gardener/landscaper/apis/deployer/utils/healthchecks/validation"
)

// ValidateProviderConfiguration validates a manifest provider configuration.
func ValidateProviderConfiguration(config *manifestv1alpha2.ProviderConfiguration) error {
	var allErrs field.ErrorList
	allErrs = append(allErrs, ValidateManifestList(field.NewPath(""), config.Manifests)...)
	allErrs = append(allErrs, ValidateTimeout(field.NewPath("deleteTimeout"), config.DeleteTimeout)...)
	allErrs = append(allErrs, ValidateTimeout(field.NewPath("healthChecks", "timeout"), config.HealthChecks.Timeout)...)
	allErrs = append(allErrs, healthchecks.ValidateHealthCheckConfiguration(field.NewPath(""), &config.HealthChecks)...)
	return allErrs.ToAggregate()
}

// ValidateManifestList validates a list of manifests.
func ValidateManifestList(fldPath *field.Path, list []manifestv1alpha2.Manifest) field.ErrorList {
	var allErrs field.ErrorList
	for i, m := range list {
		allErrs = append(allErrs, ValidateManifest(fldPath.Index(i), m)...)
	}
	return allErrs
}

// ValidateManifest validates a manifest.
func ValidateManifest(fldPath *field.Path, manifest manifestv1alpha2.Manifest) field.ErrorList {
	var allErrs field.ErrorList
	if manifest.Manifest == nil || len(manifest.Manifest.Raw) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("manifest"), "manifest must be defined"))
	}
	return allErrs
}

// ValidateTimeout validates a timeout.
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
