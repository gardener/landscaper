// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/apis/deployer/manifest"
)

// ValidateProviderConfiguration validates a manifest provider configuration.
func ValidateProviderConfiguration(config *manifest.ProviderConfiguration) error {
	var allErrs field.ErrorList
	allErrs = append(allErrs, ValidateManifestList(field.NewPath(""), config.Manifests)...)
	allErrs = append(allErrs, ValidateTimeout(field.NewPath("deleteTimeout"), config.DeleteTimeout)...)
	allErrs = append(allErrs, ValidateTimeout(field.NewPath("healthChecks", "timeout"), config.HealthChecks.Timeout)...)
	return allErrs.ToAggregate()
}

// ValidateManifestList validates a list of manifests.
func ValidateManifestList(fldPath *field.Path, list []manifest.Manifest) field.ErrorList {
	var allErrs field.ErrorList
	for i, m := range list {
		allErrs = append(allErrs, ValidateManifest(fldPath.Index(i), m)...)
	}
	return allErrs
}

// ValidateManifest validates a manifest.
func ValidateManifest(fldPath *field.Path, manifest manifest.Manifest) field.ErrorList {
	var allErrs field.ErrorList
	if manifest.Manifest == nil || len(manifest.Manifest.Raw) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("manifest"), "manifest must be defined"))
	}
	return allErrs
}

// ValidateTimeout validates a timeout.
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
