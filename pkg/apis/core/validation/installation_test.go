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
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/pkg/apis/core"
	"github.com/gardener/landscaper/pkg/apis/core/validation"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validation Testing")
}

var _ = Describe("Installation", func() {
	Context("ObjectReference", func() {
		It("should pass if ObjectReference is valid", func() {
			or := core.ObjectReference{
				Name:      "foo",
				Namespace: "bar",
			}

			allErrs := validation.ValidateObjectReference(or, field.NewPath("component"))
			Expect(allErrs).To(HaveLen(0))
		})

		It("should fail if ObjectReference.name is empty", func() {
			or := core.ObjectReference{
				Name:      "",
				Namespace: "bar",
			}

			allErrs := validation.ValidateObjectReference(or, field.NewPath("component"))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("component.name"),
			}))))
		})

		It("should fail if ObjectReference.namespace is empty", func() {
			or := core.ObjectReference{
				Name:      "foo",
				Namespace: "",
			}

			allErrs := validation.ValidateObjectReference(or, field.NewPath("component"))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("component.namespace"),
			}))))
		})
	})

	Context("ObjectReferenceList", func() {
		It("should fail if ObjectReferenceList contains invalid ObjectReferences", func() {
			orl := []core.ObjectReference{
				{
					Name:      "foo",
					Namespace: "bar",
				},
				{
					Name:      "",
					Namespace: "bar",
				},
				{
					Name:      "foo",
					Namespace: "",
				},
			}

			allErrs := validation.ValidateObjectReferenceList(orl, field.NewPath("component"))
			Expect(allErrs).ToNot(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Field": HavePrefix("component[0]"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("component[1].name"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("component[2].namespace"),
			}))))
		})
	})

	Context("InstallationImports", func() {
		It("should pass if imports are valid", func() {
			imp := core.InstallationImports{
				Data: []core.DataImport{
					{
						Name:    "foo",
						DataRef: "fooRef",
					},
					{
						Name: "bar",
						SecretRef: &core.SecretReference{
							ObjectReference: core.ObjectReference{
								Name:      "mysecret",
								Namespace: "default",
							},
							Key: "config",
						},
					},
					{
						Name: "foobar",
						ConfigMapRef: &core.ConfigMapReference{
							ObjectReference: core.ObjectReference{
								Name:      "myconfigmap",
								Namespace: "default",
							},
							Key: "config",
						},
					},
				},
				Targets: []core.TargetImportExport{{
					Name:   "foo",
					Target: "fooTarget",
				}},
			}

			allErrs := validation.ValidateInstallationImports(imp, field.NewPath("imports"))
			Expect(allErrs).To(HaveLen(0))
		})

		It("should fail if imports contain duplicate values", func() {
			imp := core.InstallationImports{
				Data: []core.DataImport{
					{
						Name:    "foo",
						DataRef: "fooRef",
					},
					{
						Name:    "foo",
						DataRef: "bar",
					},
				},
				Targets: []core.TargetImportExport{
					{
						Name:   "bar",
						Target: "barTarget",
					},
					{
						Name:   "bar",
						Target: "foo",
					},
				},
			}

			allErrs := validation.ValidateInstallationImports(imp, field.NewPath("imports"))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeDuplicate),
				"Field": Equal("imports.data[1]"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeDuplicate),
				"Field": Equal("imports.targets[1]"),
			}))))
		})

		It("should fail if imports contain empty values", func() {
			imp := core.InstallationImports{
				Data: []core.DataImport{
					{
						Name:    "",
						DataRef: "",
					},
				},
				Targets: []core.TargetImportExport{
					{
						Name:   "",
						Target: "",
					},
				},
			}

			allErrs := validation.ValidateInstallationImports(imp, field.NewPath("imports"))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("imports.data[0].name"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("imports.data[0]"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("imports.targets[0].name"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("imports.targets[0].target"),
			}))))
		})

		It("should fail if secret imports contain empty values", func() {
			imp := core.InstallationImports{
				Data: []core.DataImport{
					{
						Name:      "imp",
						SecretRef: &core.SecretReference{},
					},
				},
			}

			allErrs := validation.ValidateInstallationImports(imp, field.NewPath("imports"))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("imports.data[0].secretRef.name"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("imports.data[0].secretRef.namespace"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("imports.data[0].secretRef.key"),
			}))))
		})

		It("should fail if secret imports contain empty values", func() {
			imp := core.InstallationImports{
				Data: []core.DataImport{
					{
						Name:         "imp",
						ConfigMapRef: &core.ConfigMapReference{},
					},
				},
			}

			allErrs := validation.ValidateInstallationImports(imp, field.NewPath("imports"))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("imports.data[0].configMapRef.name"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("imports.data[0].configMapRef.namespace"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("imports.data[0].configMapRef.key"),
			}))))
		})

		It("should fail if a secret and a configmap is defined for the same import", func() {
			imp := core.InstallationImports{
				Data: []core.DataImport{
					{
						Name:         "imp",
						SecretRef:    &core.SecretReference{},
						ConfigMapRef: &core.ConfigMapReference{},
					},
				},
			}

			allErrs := validation.ValidateInstallationImports(imp, field.NewPath("imports"))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeForbidden),
				"Field": Equal("imports.data[0].secretRef"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeForbidden),
				"Field": Equal("imports.data[0].configMapRef"),
			}))))
		})
	})
})
