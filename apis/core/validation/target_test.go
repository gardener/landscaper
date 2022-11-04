// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/apis/core"
	"github.com/gardener/landscaper/apis/core/validation"
)

var _ = Describe("Target", func() {
	Context("Spec", func() {

		It("should accept a Target with an empty spec", func() {
			t := &core.Target{
				Spec: core.TargetSpec{},
			}

			allErrs := validation.ValidateTarget(t)
			Expect(allErrs).To(BeEmpty())
		})

		It("should reject a Target with secretRef and config set", func() {
			t := &core.Target{
				Spec: core.TargetSpec{
					Configuration: core.NewAnyJSONPointer([]byte("foo")),
					SecretRef: &core.SecretReference{
						ObjectReference: core.ObjectReference{
							Name:      "foo",
							Namespace: "bar",
						},
					},
				},
			}

			allErrs := validation.ValidateTarget(t)
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec"),
			}))))
		})

		It("should accept a Target with a secretRef", func() {
			t := &core.Target{
				Spec: core.TargetSpec{
					SecretRef: &core.SecretReference{
						ObjectReference: core.ObjectReference{
							Name:      "foo",
							Namespace: "bar",
						},
					},
				},
			}

			allErrs := validation.ValidateTarget(t)
			Expect(allErrs).To(BeEmpty())
		})

		It("should accept a Target with an inline config", func() {
			t := &core.Target{
				Spec: core.TargetSpec{
					Configuration: core.NewAnyJSONPointer([]byte("foo")),
				},
			}

			allErrs := validation.ValidateTarget(t)
			Expect(allErrs).To(BeEmpty())
		})

	})
})
