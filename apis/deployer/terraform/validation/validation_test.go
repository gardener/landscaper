// Copyright 2021 Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/apis/deployer/terraform"
	"github.com/gardener/landscaper/apis/deployer/terraform/validation"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "validation Test Suite")
}

var _ = Describe("Validation", func() {

	Context("Files", func() {

		It("should fail when no name is defined", func() {
			files := []terraform.FileMount{{}}
			allErr := validation.ValidateFiles(field.NewPath(""), files)
			Expect(allErr).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("[][0].name"),
			}))))
		})

		It("should fail when no jsonpath is defined for a fromTarget definition", func() {
			files := []terraform.FileMount{{
				FromTarget: &terraform.FromTarget{},
			}}
			allErr := validation.ValidateFiles(field.NewPath(""), files)
			Expect(allErr).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("[][0].fromTarget.jsonPath"),
			}))))
		})

		It("should fail when the same file is defined multiple times", func() {
			files := []terraform.FileMount{
				{
					Name: "test",
				},
				{
					Name: "test",
				},
			}
			allErr := validation.ValidateFiles(field.NewPath(""), files)
			Expect(allErr).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeDuplicate),
				"Field": Equal("[][1]"),
			}))))
		})

	})

	Context("EnvVars", func() {

		It("should fail when no name is defined", func() {
			envVars := []terraform.EnvVar{{}}
			allErr := validation.ValidateEnvVars(field.NewPath(""), envVars)
			Expect(allErr).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("[][0].name"),
			}))))
		})

		It("should fail when a env var does not comply to a c identifier restriction", func() {
			envVars := []terraform.EnvVar{{
				Name: "invalid/identifier",
			}}
			allErr := validation.ValidateEnvVars(field.NewPath(""), envVars)
			Expect(allErr).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("[][0].name"),
			}))))
		})

		It("should fail when no jsonpath is defined for a fromTarget definition", func() {
			envVars := []terraform.EnvVar{{
				FromTarget: &terraform.FromTarget{},
			}}
			allErr := validation.ValidateEnvVars(field.NewPath(""), envVars)
			Expect(allErr).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("[][0].fromTarget.jsonPath"),
			}))))
		})

		It("should fail when the same file is defined multiple times", func() {
			envVars := []terraform.EnvVar{
				{
					Name: "test",
				},
				{
					Name: "test",
				},
			}
			allErr := validation.ValidateEnvVars(field.NewPath(""), envVars)
			Expect(allErr).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeDuplicate),
				"Field": Equal("[][1]"),
			}))))
		})

	})

})
