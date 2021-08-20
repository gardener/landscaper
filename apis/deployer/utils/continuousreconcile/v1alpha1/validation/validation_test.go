// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"

	crv1alpha1 "github.com/gardener/landscaper/apis/deployer/utils/continuousreconcile/v1alpha1"
	crval "github.com/gardener/landscaper/apis/deployer/utils/continuousreconcile/v1alpha1/validation"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validation Test Suite")
}

var _ = Describe("Validation", func() {

	Context("ContinuousReconcileSpec", func() {
		It("should accept if either Cron or Every is specified and valid", func() {
			cronSpec := &crv1alpha1.ContinuousReconcileSpec{
				Cron: "* * 1 1 *",
			}
			everySpec := &crv1alpha1.ContinuousReconcileSpec{
				Every: &lsv1alpha1.Duration{Duration: 5 * time.Hour},
			}
			allErrs := field.ErrorList{}
			allErrs = append(allErrs, crval.ValidateContinuousReconcileSpec(field.NewPath("cronSpec"), cronSpec)...)
			allErrs = append(allErrs, crval.ValidateContinuousReconcileSpec(field.NewPath("everySpec"), everySpec)...)
			Expect(allErrs).To(HaveLen(0))
		})

		It("should deny if both Cron and Every are specified", func() {
			cronSpec := &crv1alpha1.ContinuousReconcileSpec{
				Cron:  "* * 1 1 *",
				Every: &lsv1alpha1.Duration{Duration: 5 * time.Hour},
			}
			allErrs := crval.ValidateContinuousReconcileSpec(field.NewPath("cronSpec"), cronSpec)
			Expect(allErrs).To(HaveLen(1))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("cronSpec"),
			}))))
		})

		It("should deny if neither Cron nor Every are specified", func() {
			cronSpec := &crv1alpha1.ContinuousReconcileSpec{}
			allErrs := crval.ValidateContinuousReconcileSpec(field.NewPath("cronSpec"), cronSpec)
			Expect(allErrs).To(HaveLen(1))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("cronSpec"),
			}))))
		})

		It("should deny if the spec is invalid", func() {
			specs := []*crv1alpha1.ContinuousReconcileSpec{
				{
					Cron: "foo",
				},
				{
					Cron: "* * *",
				},
				{
					Every: &lsv1alpha1.Duration{Duration: (-3) * time.Hour},
				},
			}
			allErrs := field.ErrorList{}
			fld := field.NewPath("cronSpec")
			for i, spec := range specs {
				allErrs = append(allErrs, crval.ValidateContinuousReconcileSpec(fld.Index(i), spec)...)
			}
			Expect(allErrs).To(HaveLen(len(specs)))
			for _, err := range allErrs {
				Expect(err.Type).To(Equal(field.ErrorTypeInvalid))
			}
		})
	})

})
