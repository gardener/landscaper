// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
