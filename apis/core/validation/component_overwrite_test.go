// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
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

var _ = Describe("ComponentOverwrites", func() {

	It("should pass if a overwrite is valid", func() {
		co := &core.ComponentOverwrites{}
		co.Overwrites = core.ComponentOverwriteList{
			{
				Component: core.ComponentOverwriteReference{
					ComponentName: "comp1",
					Version:       "0.0.1",
				},
				Target: core.ComponentOverwriteReference{
					ComponentName: "ov",
					Version:       "0.0.2-dev",
				},
			},
		}

		allErrs := validation.ValidateComponentOverwrites(co)
		Expect(allErrs).To(HaveLen(0))
	})

	Context("Target", func() {
		It("should fail if target component definition does not contain a componentName nor a version", func() {
			ov := core.ComponentOverwriteReference{}
			allErrs := validation.ValidateTargetComponent(field.NewPath("b"), ov)
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("b.componentName/version"),
			}))))
		})
	})

	Context("ComponentSource", func() {
		It("should fail if source component definition does not contain a componentName", func() {
			ov := core.ComponentOverwriteReference{}
			allErrs := validation.ValidateSourceComponent(field.NewPath("b"), ov)
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("b.componentName"),
			}))))
		})
	})

})
