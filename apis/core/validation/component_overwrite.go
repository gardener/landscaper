// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/apis/core"
)

// ValidateComponentOverwrites validates ComponentOverwrites
func ValidateComponentOverwrites(co *core.ComponentOverwrites) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateComponentOverwriteList(field.NewPath("overwrites"), co.Overwrites)...)
	return allErrs
}

// ValidateComponentOverwriteList validates a list of component overwrites
func ValidateComponentOverwriteList(fldPath *field.Path, coList core.ComponentOverwriteList) field.ErrorList {
	var allErrs field.ErrorList
	for i, co := range coList {
		coPath := fldPath.Index(i)
		allErrs = append(allErrs, ValidateSourceComponent(coPath.Child("component"), co.Component)...)
		allErrs = append(allErrs, ValidateTargetComponent(coPath.Child("target"), co.Target)...)
	}
	return allErrs
}

// ValidateSourceComponent validates a component reference that should be replaced.
func ValidateSourceComponent(fldPath *field.Path, component core.ComponentOverwriteReference) field.ErrorList {
	var allErrs field.ErrorList
	if len(component.ComponentName) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("componentName"), "componentName must be defined"))
	}
	return allErrs
}

// ValidateTargetComponent validates a target component reference.
func ValidateTargetComponent(fldPath *field.Path, component core.ComponentOverwriteReference) field.ErrorList {
	var allErrs field.ErrorList
	if len(component.ComponentName) == 0 && len(component.Version) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("componentName/version"), "a componentName or a version target has to be defined"))
	}
	return allErrs
}
