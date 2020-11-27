// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"encoding/base64"

	"k8s.io/apimachinery/pkg/util/validation/field"

	terraformv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/terraform/v1alpha1"
)

// ValidateProviderConfiguration validates a terraform deployer configuration
func ValidateProviderConfiguration(config *terraformv1alpha1.ProviderConfiguration) error {
	allErrs := field.ErrorList{}
	if len(config.Kubeconfig) != 0 {
		if _, err := base64.StdEncoding.DecodeString(config.Kubeconfig); err != nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("kubeconfig"), config.Kubeconfig, "must be a valid base64 encoded string"))
		}
	}

	if len(config.Namespace) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("namespace"), "must not be empty"))
	}

	if len(config.Main) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("main.tf"), "must not be empty"))
	}

	return allErrs.ToAggregate()
}
