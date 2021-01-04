// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	terraformv1alpha1 "github.com/gardener/landscaper/apis/deployer/terraform/v1alpha1"
)

// ValidateProviderConfiguration validates a terraform deployer configuration
func ValidateProviderConfiguration(config *terraformv1alpha1.ProviderConfiguration) error {
	allErrs := field.ErrorList{}

	if len(config.Main) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("main.tf"), "must not be empty"))
	}

	return allErrs.ToAggregate()
}
