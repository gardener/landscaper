// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	cron "github.com/robfig/cron/v3"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helper "github.com/gardener/landscaper/apis/core/validation"
	cr "github.com/gardener/landscaper/apis/deployer/utils/continuousreconcile"
)

// ValidateContinuousReconcileSpec validates a continuous reconciliation spec.
// A value of nil is considered valid, as is an 'empty' spec with all fields equal to nil or their respective zero value.
func ValidateContinuousReconcileSpec(fldPath *field.Path, spec *cr.ContinuousReconcileSpec) field.ErrorList {
	if ContinuousReconcileSpecIsEmpty(spec) {
		return nil
	}
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, helper.ValidateExactlyOneOf(fldPath, spec, "Cron", "Every")...)
	if len(spec.Cron) != 0 {
		allErrs = append(allErrs, validateCronSpec(fldPath.Child("cron"), spec.Cron)...)
	}
	if spec.Every != nil {
		allErrs = append(allErrs, validateEveryDuration(fldPath.Child("every"), spec.Every)...)
	}
	return allErrs
}

func validateCronSpec(fldPath *field.Path, cronSpec string) field.ErrorList {
	_, err := cron.ParseStandard(cronSpec)
	if err != nil {
		return field.ErrorList{field.Invalid(fldPath, cronSpec, err.Error())}
	}
	return nil
}

func validateEveryDuration(fldPath *field.Path, dur *lsv1alpha1.Duration) field.ErrorList {
	if dur != nil && dur.Duration <= 0 {
		return field.ErrorList{field.Invalid(fldPath, dur, "specified duration has to be greater than zero")}
	}
	return nil
}

// ContinuousReconcileSpecIsEmpty returns true if the given spec is either nil or all its fields are nil/the zero value.
func ContinuousReconcileSpecIsEmpty(spec *cr.ContinuousReconcileSpec) bool {
	return spec == nil || (len(spec.Cron) == 0 && (spec.Every == nil || spec.Every.Duration == 0))
}
