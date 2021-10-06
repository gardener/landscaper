// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/apis/deployer/utils/managedresource/validation"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	crval "github.com/gardener/landscaper/apis/deployer/utils/continuousreconcile/validation"
	health "github.com/gardener/landscaper/apis/deployer/utils/readinesschecks/validation"
)

// ValidateProviderConfiguration validates a manifest provider configuration.
func ValidateProviderConfiguration(config *manifestv1alpha2.ProviderConfiguration) error {
	var allErrs field.ErrorList
	allErrs = append(allErrs, validation.ValidateManifestList(field.NewPath(""), config.Manifests)...)
	allErrs = append(allErrs, ValidateTimeout(field.NewPath("deleteTimeout"), config.DeleteTimeout)...)
	allErrs = append(allErrs, ValidateTimeout(field.NewPath("readinessChecks", "timeout"), config.ReadinessChecks.Timeout)...)
	allErrs = append(allErrs, health.ValidateReadinessCheckConfiguration(field.NewPath(""), &config.ReadinessChecks)...)
	allErrs = append(allErrs, crval.ValidateContinuousReconcileSpec(field.NewPath("continuousReconcile"), config.ContinuousReconcile)...)
	return allErrs.ToAggregate()
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
