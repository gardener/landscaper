// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/apis/deployer/terraform"
)

// ValidateProviderConfiguration validates a terraform deployer configuration
func ValidateProviderConfiguration(config *terraform.ProviderConfiguration) error {
	allErrs := field.ErrorList{}

	if len(config.Main) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("main.tf"), "must not be empty"))
	}

	allErrs = append(allErrs, ValidateEnvVars(field.NewPath("envVars"), config.EnvVars)...)
	allErrs = append(allErrs, ValidateFiles(field.NewPath("files"), config.Files)...)

	return allErrs.ToAggregate()
}

// ValidateFiles validate a list of file definitions.
func ValidateFiles(fldPath *field.Path, files []terraform.FileMount) field.ErrorList {
	var allErrs field.ErrorList
	names := sets.NewString()
	for i, file := range files {
		fpath := fldPath.Index(i)
		if len(file.Name) == 0 {
			allErrs = append(allErrs, field.Required(fpath.Child("name"), "must not be empty"))
		}

		if file.FromTarget != nil {
			if len(file.FromTarget.JSONPath) == 0 {
				allErrs = append(allErrs, field.Required(fpath.Child("fromTarget").Child("jsonPath"), "must be defined"))
			}
		}

		if names.Has(file.Name) {
			allErrs = append(allErrs, field.Duplicate(fpath, file))
		}
		names.Insert(file.Name)
	}

	return allErrs
}

// ValidateEnvVars validate a list of environment variable definitions.
func ValidateEnvVars(fldPath *field.Path, envvars []terraform.EnvVar) field.ErrorList {
	var allErrs field.ErrorList
	names := sets.NewString()
	for i, env := range envvars {
		fpath := fldPath.Index(i)
		if len(env.Name) == 0 {
			allErrs = append(allErrs, field.Required(fpath.Child("name"), "must not be empty"))
		} else {
			errs := validation.IsCIdentifier(env.Name)
			if len(errs) != 0 {
				// the IsCIdentifier functions returns one error mesasge if the check fails.
				allErrs = append(allErrs, field.Invalid(fpath.Child("name"), env.Name, errs[0]))
			}
		}

		if env.FromTarget != nil {
			if len(env.FromTarget.JSONPath) == 0 {
				allErrs = append(allErrs, field.Required(fpath.Child("fromTarget").Child("jsonPath"), "must be defined"))
			}
		}

		if names.Has(env.Name) {
			allErrs = append(allErrs, field.Duplicate(fpath, env))
		}
		names.Insert(env.Name)
	}

	return allErrs
}
