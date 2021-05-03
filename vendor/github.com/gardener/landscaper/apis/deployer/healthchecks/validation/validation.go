// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"errors"

	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/util/jsonpath"

	"github.com/gardener/landscaper/apis/deployer/healthchecks"
)

// ValidateHealthCheckConfiguration validates a healthcheck configuration
func ValidateHealthCheckConfiguration(fldPath *field.Path, config *healthchecks.HealthChecksConfiguration) field.ErrorList {
	var allErrs field.ErrorList

	// if we have a custom healthcheck configuration, the default should be disabled
	disableDefault := config.DisableDefault
	customHealthChecks := config.CustomHealthChecks
	if !disableDefault && customHealthChecks != nil && len(customHealthChecks) > 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("customHealthChecks"), disableDefault, "custom health checks require disabling the default health check"))
	}

	for _, c := range customHealthChecks {
		allErrs = append(allErrs, ValidateCustomHealthCheckConfiguration(fldPath.Child("customHealthCheckConfiguration"), &c)...)
	}

	return allErrs
}

// ValidateCustomHealthCheckConfiguration validates a custom healthcheck configuration
func ValidateCustomHealthCheckConfiguration(fldPath *field.Path, config *healthchecks.CustomHealthCheckConfiguration) field.ErrorList {
	var allErrs field.ErrorList
	if len(config.Name) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "must not be empty"))
	}

	if config.Resource == nil && config.LabelSelector == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("resourceSelector/labelSelector"), "must have either a resource or a set of labels"))
	}

	if config.LabelSelector != nil {
		allErrs = append(allErrs, ValidateLabelSelectorSpec(fldPath.Child("labelSelector"), config.LabelSelector)...)
	}

	if len(config.Requirements) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("requirements"), "must have at least one resource requirement"))

	}

	for _, r := range config.Requirements {
		allErrs = append(allErrs, ValidateRequirementSpec(fldPath.Child("requirements"), &r)...)
	}

	return allErrs
}

// ValidateRequirementSpec validates a requirement specification for a custom healthcheck configuration
func ValidateRequirementSpec(fldPath *field.Path, spec *healthchecks.RequirementSpec) field.ErrorList {
	var allErrs field.ErrorList

	if len(spec.JsonPath) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("jsonPath"), "JSON path must be defined"))
	}

	p := jsonpath.New("test")
	err := p.Parse(spec.JsonPath)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("jsonPath"), spec.JsonPath, "is not a valid JSON path template"))
	}

	op := spec.Operator
	err = validateOperator(op)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("operator"), op, "is not a valid operator"))
	}

	if op != selection.DoesNotExist && op != selection.Exists && len(spec.Value) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("value"), "must have at least one value to compare to"))
	}

	if op != selection.In && op != selection.NotIn && len(spec.Value) > 1 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("value"), op, "must have exactly one value to compare to"))
	}

	return allErrs
}

// ValidateLabelSelectorSpec validates a LabelSelector specification for a custom healthcheck configuration
func ValidateLabelSelectorSpec(fldPath *field.Path, spec *healthchecks.LabelSelectorSpec) field.ErrorList {
	var allErrs field.ErrorList

	if len(spec.Labels) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("matchLabels"), "must have at least one label to select on"))
	}

	return allErrs
}

func validateOperator(operator selection.Operator) error {
	var allowedOperators = []selection.Operator{
		selection.DoesNotExist,
		selection.Exists,
		selection.Equals,
		selection.DoubleEquals,
		selection.NotEquals,
		selection.In,
		selection.NotIn,
	}

	for _, op := range allowedOperators {
		if operator == op {
			return nil
		}
	}
	return errors.New("not an allowed operator")
}
