// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/pkg/apis/core"
	"github.com/gardener/landscaper/pkg/apis/core/validation"
)

var _ = Describe("DeployItem", func() {

	Context("Spec", func() {
		It("should pass if a DeployItem spec is valid (without target)", func() {
			diSpec := core.DeployItemSpec{}
			diSpec.Type = "foo"

			allErrs := validation.ValidateDeployItemSpec(field.NewPath(""), diSpec)
			Expect(allErrs).To(HaveLen(0))
		})

		It("should pass if a DeployItem spec is valid (with target)", func() {
			diSpec := core.DeployItemSpec{}
			diSpec.Type = "foo"
			diSpec.Target = &core.ObjectReference{}
			diSpec.Target.Name = "bar"
			diSpec.Target.Namespace = "baz"

			allErrs := validation.ValidateDeployItemSpec(field.NewPath(""), diSpec)
			Expect(allErrs).To(HaveLen(0))
		})

		It("should fail if a DeployItem spec is invalid (empty type)", func() {
			diSpec := core.DeployItemSpec{}
			diSpec.Type = ""

			allErrs := validation.ValidateDeployItemSpec(field.NewPath("di"), diSpec)
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("di.type"),
			}))))
		})

		It("should fail if a DeployItem spec is invalid (empty target)", func() {
			diSpec := core.DeployItemSpec{}
			diSpec.Type = "foo"
			diSpec.Target = &core.ObjectReference{}
			diSpec.Target.Name = ""
			diSpec.Target.Namespace = ""

			allErrs := validation.ValidateDeployItemSpec(field.NewPath("di"), diSpec)
			Expect(allErrs).To(HaveLen(2))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("di.target.name"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("di.target.namespace"),
			}))))
		})
	})

})
