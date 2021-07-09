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

	"github.com/gardener/landscaper/apis/deployer/utils/readinesschecks"
	"github.com/gardener/landscaper/apis/deployer/utils/readinesschecks/validation"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "validation Test Suite")
}

var _ = Describe("Validation", func() {

	var (
		fld *field.Path

		rc      readinesschecks.ReadinessCheckConfiguration
		cc      readinesschecks.CustomReadinessCheckConfiguration
		reqSpec readinesschecks.RequirementSpec
	)

	BeforeEach(func() {
		fld = field.NewPath("readinessCheck")

		reqSpec = readinesschecks.RequirementSpec{
			JsonPath: ".foo.bar",
			Operator: selection.Equals,
			Value: []runtime.RawExtension{
				{
					Raw: []byte("foobar"),
				},
			},
		}

		cc = readinesschecks.CustomReadinessCheckConfiguration{
			Name:     "customReadinessCheck",
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
			LabelSelector: &readinesschecks.LabelSelectorSpec{
				APIVersion: "v1",
				Kind:       "Service",
				Labels: map[string]string{
					"foo": "bar",
				},
			},
			Requirements: []readinesschecks.RequirementSpec{reqSpec},
		}

		rc = readinesschecks.ReadinessCheckConfiguration{
			DisableDefault:        true,
			Timeout:               &lsv1alpha1.Duration{Duration: 180 * time.Second},
			CustomReadinessChecks: []readinesschecks.CustomReadinessCheckConfiguration{cc},
		}

	})

	It("should accept a readiness check configuration with default readiness check disabled and custom readiness check present", func() {
		allErrs := validation.ValidateReadinessCheckConfiguration(fld, &rc)
		Expect(allErrs).To(HaveLen(0))
	})

	It("should accept a readiness check configuration with default readiness check enabled and custom readiness check present", func() {
		rc.DisableDefault = false

		allErrs := validation.ValidateReadinessCheckConfiguration(fld, &rc)
		Expect(allErrs).To(HaveLen(0))
	})

	It("should accept a provider configuration with default readiness checks enabled and no custom readiness check present", func() {
		rc.DisableDefault = false
		rc.CustomReadinessChecks = nil

		allErrs := validation.ValidateReadinessCheckConfiguration(fld, &rc)
		Expect(allErrs).To(HaveLen(0))
	})

	It("should reject a custom readiness check without a name", func() {
		rc.CustomReadinessChecks[0].Name = ""

		allErrs := validation.ValidateReadinessCheckConfiguration(fld, &rc)
		Expect(allErrs).To(HaveLen(1))
	})

	It("should reject a custom readiness check with no labelselector and no resourceselector", func() {
		rc.CustomReadinessChecks[0].LabelSelector = nil
		rc.CustomReadinessChecks[0].Resource = nil

		allErrs := validation.ValidateReadinessCheckConfiguration(fld, &rc)
		Expect(allErrs).To(HaveLen(1))
	})

	It("should reject a custom readiness check with a labelselector but no labels", func() {
		rc.CustomReadinessChecks[0].LabelSelector.Labels = nil

		allErrs := validation.ValidateReadinessCheckConfiguration(fld, &rc)
		Expect(allErrs).To(HaveLen(1))
	})

	It("should accept a custom readiness check with no resource but with a labelselector", func() {
		rc.CustomReadinessChecks[0].Resource = nil

		allErrs := validation.ValidateReadinessCheckConfiguration(fld, &rc)
		Expect(allErrs).To(HaveLen(0))
	})

	It("should accept a custom readiness check with no resource but with a labelselector", func() {
		rc.CustomReadinessChecks[0].Resource = nil

		allErrs := validation.ValidateReadinessCheckConfiguration(fld, &rc)
		Expect(allErrs).To(HaveLen(0))
	})

	It("should reject a custom readiness check with no requirements", func() {
		rc.CustomReadinessChecks[0].Requirements = nil

		allErrs := validation.ValidateReadinessCheckConfiguration(fld, &rc)
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
			rc.CustomReadinessChecks[0].Requirements[0].Operator = op
			errs := validation.ValidateReadinessCheckConfiguration(fld, &rc)
			allErrs = append(allErrs, errs...)
		}
		Expect(allErrs).To(HaveLen(0))

		var disAllowedOperators = []selection.Operator{
			selection.GreaterThan,
			selection.LessThan,
			"completelyInvalidOperatorExpression",
		}

		for _, op := range disAllowedOperators {
			rc.CustomReadinessChecks[0].Requirements[0].Operator = op
			errs := validation.ValidateReadinessCheckConfiguration(fld, &rc)
			allErrs = append(allErrs, errs...)
		}
		Expect(allErrs).To(HaveLen(3))
	})

	It("should reject an empty JSON path in a requirement spec", func() {
		rc.CustomReadinessChecks[0].Requirements[0].JsonPath = ""
		allErrs := validation.ValidateReadinessCheckConfiguration(fld, &rc)
		Expect(allErrs).To(HaveLen(1))
	})

	It("should reject empty values in a requirement spec", func() {
		rc.CustomReadinessChecks[0].Requirements[0].Value = nil
		allErrs := validation.ValidateReadinessCheckConfiguration(fld, &rc)
		Expect(allErrs).To(HaveLen(1))
	})

	It("should reject a requirement spec with 'equals/double-equals/not-equals' operator and multiple values", func() {
		var allErrs field.ErrorList
		var operators = []selection.Operator{
			selection.Equals,
			selection.DoubleEquals,
			selection.NotEquals,
		}

		val := rc.CustomReadinessChecks[0].Requirements[0].Value
		val = append(val, runtime.RawExtension{Raw: []byte("johndoe")})
		rc.CustomReadinessChecks[0].Requirements[0].Value = val

		for _, op := range operators {
			rc.CustomReadinessChecks[0].Requirements[0].Operator = op
			errs := validation.ValidateReadinessCheckConfiguration(fld, &rc)
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

		val := rc.CustomReadinessChecks[0].Requirements[0].Value
		val = append(val, runtime.RawExtension{Raw: []byte("johndoe")})
		rc.CustomReadinessChecks[0].Requirements[0].Value = val

		for _, op := range operators {
			rc.CustomReadinessChecks[0].Requirements[0].Operator = op
			errs := validation.ValidateReadinessCheckConfiguration(fld, &rc)
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

		rc.CustomReadinessChecks[0].Requirements[0].Value = nil

		for _, op := range operators {
			rc.CustomReadinessChecks[0].Requirements[0].Operator = op
			errs := validation.ValidateReadinessCheckConfiguration(fld, &rc)
			allErrs = append(allErrs, errs...)
		}
		Expect(allErrs).To(HaveLen(0))
	})
})
