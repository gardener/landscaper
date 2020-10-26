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

var _ = Describe("Execution", func() {

	Context("ValidateDeployItemTemplate", func() {
		It("should pass if a DeployItemTemplate is valid", func() {
			tmpl := core.DeployItemTemplate{}
			tmpl.Name = "my-import"
			tmpl.Type = "mytype"

			allErrs := validation.ValidateDeployItemTemplate(field.NewPath(""), tmpl)
			Expect(allErrs).To(HaveLen(0))
		})

		It("should fail if DeployItemTemplate.name is empty", func() {
			tmpl := core.DeployItemTemplate{}

			allErrs := validation.ValidateDeployItemTemplate(field.NewPath("b"), tmpl)
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("b.name"),
			}))))
		})

		It("should fail if DeployItemTemplate.type is empty", func() {
			tmpl := core.DeployItemTemplate{}

			allErrs := validation.ValidateDeployItemTemplate(field.NewPath("b"), tmpl)
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("b.type"),
			}))))
		})
	})

	Context("ValidateDeployItemTemplateList", func() {
		It("should pass if a DeployItemTemplate is valid", func() {
			tmpl := core.DeployItemTemplate{}
			tmpl.Name = "my-import"
			tmpl.Type = "mytype"

			allErrs := validation.ValidateDeployItemTemplateList(field.NewPath(""), []core.DeployItemTemplate{tmpl})
			Expect(allErrs).To(HaveLen(0))
		})

		It("should fail if duplicated DeployItemTemplates are defined", func() {
			templates := []core.DeployItemTemplate{
				{
					Name: "test",
				},
				{
					Name: "test",
				},
			}

			allErrs := validation.ValidateDeployItemTemplateList(field.NewPath("b"), templates)
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":     Equal(field.ErrorTypeDuplicate),
				"Field":    Equal("b[1]"),
				"BadValue": Equal("test"),
			}))))
		})

	})

})
