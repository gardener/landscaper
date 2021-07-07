// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/apis/deployer/utils/healthchecks"
	"github.com/gardener/landscaper/apis/deployer/utils/healthchecks/validation"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "validation Test Suite")
}

var _ = Describe("Validation", func() {

	var (
		fld *field.Path

		hc healthchecks.HealthChecksConfiguration
		cc healthchecks.CustomHealthCheckConfiguration
		rc healthchecks.RequirementSpec
	)

	BeforeEach(func() {
		fld = field.NewPath("healthcheck")

		rc = healthchecks.RequirementSpec{
			JsonPath: "{ .foo.bar }",
			Operator: selection.Equals,
			Value: []runtime.RawExtension{
				{
					Raw: []byte("foobar"),
				},
			},
		}

		cc = healthchecks.CustomHealthCheckConfiguration{
			Name:     "customHealthCheck",
			Timeout:  nil,
			Disabled: false,
			Resource: []lsv1alpha1.TypedObjectReference{
				{
					APIVersion: "v1",
					Kind:       "Service",
					ObjectReference: lsv1alpha1.ObjectReference{
						Name:      "foo",
						Namespace: "bar",
					},
				},
			},
			LabelSelector: &healthchecks.LabelSelectorSpec{
				APIVersion: "v1",
				Kind:       "Service",
				Labels: map[string]string{
					"foo": "bar",
				},
			},
			Requirements: []healthchecks.RequirementSpec{rc},
		}

		hc = healthchecks.HealthChecksConfiguration{
			DisableDefault:     true,
			Timeout:            &lsv1alpha1.Duration{Duration: 180 * time.Second},
			CustomHealthChecks: []healthchecks.CustomHealthCheckConfiguration{cc},
		}

	})

	It("should accept a health check configuration with default health check disabled and custom healthchecks present", func() {
		allErrs := validation.ValidateHealthCheckConfiguration(fld, &hc)
		Expect(allErrs).To(HaveLen(0))
	})

	It("should accept a health check configuration with default health check enabled and custom healthchecks present", func() {
		hc.DisableDefault = false

		allErrs := validation.ValidateHealthCheckConfiguration(fld, &hc)
		Expect(allErrs).To(HaveLen(0))
	})

	It("should accept a provider configuration with default Health checks enabled and no custom healthchecks present", func() {
		hc.DisableDefault = false
		hc.CustomHealthChecks = nil

		allErrs := validation.ValidateHealthCheckConfiguration(fld, &hc)
		Expect(allErrs).To(HaveLen(0))
	})

	It("should reject a custom healthCheck without a name", func() {
		hc.CustomHealthChecks[0].Name = ""

		allErrs := validation.ValidateHealthCheckConfiguration(fld, &hc)
		Expect(allErrs).To(HaveLen(1))
	})

	It("should reject a custom healthCheck with no labelselector and no resourceselector", func() {
		hc.CustomHealthChecks[0].LabelSelector = nil
		hc.CustomHealthChecks[0].Resource = nil

		allErrs := validation.ValidateHealthCheckConfiguration(fld, &hc)
		Expect(allErrs).To(HaveLen(1))
	})

	It("should reject a custom healthCheck with a labelselector but no labels", func() {
		hc.CustomHealthChecks[0].LabelSelector.Labels = nil

		allErrs := validation.ValidateHealthCheckConfiguration(fld, &hc)
		Expect(allErrs).To(HaveLen(1))
	})

	It("should accept a custom healthCheck with no resource but with a labelselector", func() {
		hc.CustomHealthChecks[0].Resource = nil

		allErrs := validation.ValidateHealthCheckConfiguration(fld, &hc)
		Expect(allErrs).To(HaveLen(0))
	})

	It("should accept a custom healthCheck with no resource but with a labelselector", func() {
		hc.CustomHealthChecks[0].Resource = nil

		allErrs := validation.ValidateHealthCheckConfiguration(fld, &hc)
		Expect(allErrs).To(HaveLen(0))
	})

	It("should reject a custom healthCheck with no requirements", func() {
		hc.CustomHealthChecks[0].Requirements = nil

		allErrs := validation.ValidateHealthCheckConfiguration(fld, &hc)
		Expect(allErrs).To(HaveLen(1))
	})

	It("should only accept allowed operators in a requirement spec", func() {
		var allErrs field.ErrorList

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
			hc.CustomHealthChecks[0].Requirements[0].Operator = op
			errs := validation.ValidateHealthCheckConfiguration(fld, &hc)
			allErrs = append(allErrs, errs...)
		}
		Expect(allErrs).To(HaveLen(0))

		var disAllowedOperators = []selection.Operator{
			selection.GreaterThan,
			selection.LessThan,
			"completelyInvalidOperatorExpression",
		}

		for _, op := range disAllowedOperators {
			hc.CustomHealthChecks[0].Requirements[0].Operator = op
			errs := validation.ValidateHealthCheckConfiguration(fld, &hc)
			allErrs = append(allErrs, errs...)
		}
		Expect(allErrs).To(HaveLen(3))
	})

	It("should reject an empty JSON path in a requirement spec", func() {
		hc.CustomHealthChecks[0].Requirements[0].JsonPath = ""
		allErrs := validation.ValidateHealthCheckConfiguration(fld, &hc)
		Expect(allErrs).To(HaveLen(1))
	})

	It("should reject empty values in a requirement spec", func() {
		hc.CustomHealthChecks[0].Requirements[0].Value = nil
		allErrs := validation.ValidateHealthCheckConfiguration(fld, &hc)
		Expect(allErrs).To(HaveLen(1))
	})

	It("should reject a requirement spec with 'equals/double-equals/not-equals' operator and multiple values", func() {
		var allErrs field.ErrorList
		var operators = []selection.Operator{
			selection.Equals,
			selection.DoubleEquals,
			selection.NotEquals,
		}

		val := hc.CustomHealthChecks[0].Requirements[0].Value
		val = append(val, runtime.RawExtension{Raw: []byte("johndoe")})
		hc.CustomHealthChecks[0].Requirements[0].Value = val

		for _, op := range operators {
			hc.CustomHealthChecks[0].Requirements[0].Operator = op
			errs := validation.ValidateHealthCheckConfiguration(fld, &hc)
			allErrs = append(allErrs, errs...)
		}

		Expect(allErrs).To(HaveLen(3))
	})

	It("should accept a requirement spec with 'in/notin' operator and multiple values", func() {
		var allErrs field.ErrorList
		var operators = []selection.Operator{
			selection.In,
			selection.NotIn,
		}

		val := hc.CustomHealthChecks[0].Requirements[0].Value
		val = append(val, runtime.RawExtension{Raw: []byte("johndoe")})
		hc.CustomHealthChecks[0].Requirements[0].Value = val

		for _, op := range operators {
			hc.CustomHealthChecks[0].Requirements[0].Operator = op
			errs := validation.ValidateHealthCheckConfiguration(fld, &hc)
			allErrs = append(allErrs, errs...)
		}

		Expect(allErrs).To(HaveLen(0))
	})

	It("should accept empty values in a requirement spec with 'in/not-in' operators", func() {
		var allErrs field.ErrorList
		var operators = []selection.Operator{
			selection.DoesNotExist,
			selection.Exists,
		}

		hc.CustomHealthChecks[0].Requirements[0].Value = nil

		for _, op := range operators {
			hc.CustomHealthChecks[0].Requirements[0].Operator = op
			errs := validation.ValidateHealthCheckConfiguration(fld, &hc)
			allErrs = append(allErrs, errs...)
		}
		Expect(allErrs).To(HaveLen(0))
	})
})
