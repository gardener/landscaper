// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/validation"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
)

// ValidateProviderConfiguration validates a container deployer configuration
func ValidateProviderConfiguration(config *containerv1alpha1.ProviderConfiguration) error {
	var allErrs field.ErrorList
	for i, secretRef := range config.RegistryPullSecrets {
		coreSecretRef := core.ObjectReference{}
		if err := lsv1alpha1.Convert_v1alpha1_ObjectReference_To_core_ObjectReference(&secretRef, &coreSecretRef, nil); err != nil {
			return err
		}
		allErrs = append(allErrs, validation.ValidateObjectReference(coreSecretRef, field.NewPath("registryPullSecrets").Index(i))...)
	}

	return allErrs.ToAggregate()
}
