// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/apis/core"
)

// ValidateDeployItem validates a DeployItem
func ValidateDeployItem(di *core.DeployItem) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateDeployItemSpec(field.NewPath("spec"), di.Spec)...)

	return allErrs
}

// ValidateDeployItemSpec validates a DeployItem spec
func ValidateDeployItemSpec(fldPath *field.Path, diSpec core.DeployItemSpec) field.ErrorList {
	allErrs := field.ErrorList{}

	if diSpec.Type == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("type"), "type must not be empty"))
	}
	if diSpec.Target != nil {
		if diSpec.Target.Name == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("target").Child("name"), "target name must not be empty"))
		}
		if diSpec.Target.Namespace == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("target").Child("namespace"), "target namespace must not be empty"))
		}
	}

	return allErrs
}
