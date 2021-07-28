// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	"github.com/gardener/landscaper/apis/core"
	"github.com/gardener/landscaper/apis/core/validation"
)

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

	Context("InstallationBlueprint", func() {
		It("should accept a Blueprint reference", func() {
			bpDef := core.BlueprintDefinition{
				Reference: &core.RemoteBlueprintReference{
					ResourceName: "foo",
				},
				Inline: nil,
			}
			allErrs := validation.ValidateInstallationBlueprint(bpDef, field.NewPath("blueprint"))
			Expect(allErrs).To(HaveLen(0))
		})

		It("should accept an inline Blueprint", func() {
			bpDef := core.BlueprintDefinition{
				Reference: nil,
				Inline: &core.InlineBlueprint{
					Filesystem: core.AnyJSON{RawMessage: []byte("raw-string-representing-inline-blueprint-for-test")},
				},
			}
			allErrs := validation.ValidateInstallationBlueprint(bpDef, field.NewPath("blueprint"))
			Expect(allErrs).To(HaveLen(0))
		})

		It("should reject empty Blueprint reference and inline definition to be nil at the same time", func() {
			bpDef := core.BlueprintDefinition{
				Reference: nil,
				Inline:    nil,
			}
			allErrs := validation.ValidateInstallationBlueprint(bpDef, field.NewPath("blueprint"))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("blueprint.definition"),
			}))))
		})

		It("should reject Blueprint reference and inline definition to be given at the same time", func() {
			bpDef := core.BlueprintDefinition{
				Reference: &core.RemoteBlueprintReference{
					ResourceName: "foo",
				},
				Inline: &core.InlineBlueprint{
					Filesystem: core.AnyJSON{RawMessage: []byte("raw-string-representing-inline-blueprint-for-test")},
				},
			}
			allErrs := validation.ValidateInstallationBlueprint(bpDef, field.NewPath("blueprint"))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("blueprint.definition"),
			}))))
		})
	})

	Context("InstallationComponentDescriptor", func() {
		It("should accept a nil ComponentDescriptor definition", func() {
			var cdDef *core.ComponentDescriptorDefinition = nil

			allErrs := validation.ValidateInstallationComponentDescriptor(cdDef, field.NewPath("componentDescriptor"))
			Expect(allErrs).To(HaveLen(0))
		})

		It("should accept a ComponentDescriptor reference", func() {
			repoCtx, _ := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("http://foo.invalid", ""))
			cdDef := &core.ComponentDescriptorDefinition{
				Reference: &core.ComponentDescriptorReference{
					RepositoryContext: &repoCtx,
					ComponentName:     "foo",
					Version:           "123",
				},
				Inline: nil,
			}

			allErrs := validation.ValidateInstallationComponentDescriptor(cdDef, field.NewPath("componentDescriptor"))
			Expect(allErrs).To(HaveLen(0))
		})

		It("should accept an inline ComponentDescriptor", func() {
			cdDef := &core.ComponentDescriptorDefinition{
				Reference: nil,
				Inline:    &cdv2.ComponentDescriptor{},
			}

			allErrs := validation.ValidateInstallationComponentDescriptor(cdDef, field.NewPath("componentDescriptor"))
			Expect(allErrs).To(HaveLen(0))
		})

		It("should reject ComponentDescriptor reference and inline definition to be nil at the same time", func() {
			cdDef := &core.ComponentDescriptorDefinition{
				Reference: nil,
				Inline:    nil,
			}

			allErrs := validation.ValidateInstallationComponentDescriptor(cdDef, field.NewPath("componentDescriptor"))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("componentDescriptor.definition"),
			}))))
		})

		It("should reject ComponentDescriptor reference and inline definition to be given at the same time", func() {
			repoCtx, _ := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("http://foo.invalid", ""))
			cdDef := &core.ComponentDescriptorDefinition{
				Reference: &core.ComponentDescriptorReference{
					RepositoryContext: &repoCtx,
					ComponentName:     "foo",
					Version:           "123",
				},
				Inline: &cdv2.ComponentDescriptor{},
			}

			allErrs := validation.ValidateInstallationComponentDescriptor(cdDef, field.NewPath("componentDescriptor"))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("componentDescriptor.definition"),
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
				Targets: []core.TargetImport{
					{
						Name:   "foobaz",
						Target: "fooTarget",
					},
					{
						Name:                "barbaz",
						TargetListReference: "foobaz",
					},
					{
						Name: "baz",
						Targets: []string{
							"t1",
							"t2",
							"t3",
						},
					},
					{
						Name:    "fb",
						Targets: []string{},
					},
				},
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
				Targets: []core.TargetImport{
					{
						Name:   "bar",
						Target: "barTarget",
					},
					{
						Name:   "bar",
						Target: "foo",
					},
					{
						Name:   "foo",
						Target: "foo",
					},
				},
			}

			allErrs := validation.ValidateInstallationImports(imp, field.NewPath("imports"))
			Expect(allErrs).To(HaveLen(3))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeDuplicate),
				"Field": Equal("imports.data[1]"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeDuplicate),
				"Field": Equal("imports.targets[1]"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeDuplicate),
				"Field": Equal("imports.targets[2]"),
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
				Targets: []core.TargetImport{
					{
						Name:   "",
						Target: "",
					},
					{
						Name: "foo",
						Targets: []string{
							"",
						},
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
				"Field": Equal("imports.targets[0].target|targets|targetListRef"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("imports.targets[1].targets[0]"),
			}))))
		})

		It("should fail if multiple of target, targets, and targetListReference are specified", func() {
			imp := core.InstallationImports{
				Targets: []core.TargetImport{
					{
						Name:   "foo",
						Target: "foobar",
						Targets: []string{
							"bar",
						},
					},
					{
						Name:                "bar",
						Target:              "foobar",
						TargetListReference: "foobaz",
					},
					{
						Name:                "baz",
						Targets:             []string{},
						TargetListReference: "foobaz",
					},
				},
			}

			allErrs := validation.ValidateInstallationImports(imp, field.NewPath("imports"))
			Expect(allErrs).To(HaveLen(3))
			for _, elem := range allErrs {
				Expect(elem).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Detail": Equal("only one of target, targets, and targetListRef may be specified"),
				})))
			}
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
