// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource/validation"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "validation Test Suite")
}

var _ = Describe("Validation", func() {

	var (
		fld = field.NewPath("a")
	)

	Context("Export", func() {
		It("should accept if a key and a jsonpath is set", func() {
			export := &managedresource.Export{
				Key:      "abc",
				JSONPath: "b",
			}
			allErrs := validation.ValidateManifestExport(fld, export)
			Expect(allErrs).To(HaveLen(0))
		})

		It("should accept if a valid resource is referenced", func() {
			export := &managedresource.Export{
				Key:      "abc",
				JSONPath: "b",
				FromResource: &lsv1alpha1.TypedObjectReference{
					APIVersion: "v1",
					Kind:       "Secret",
					ObjectReference: lsv1alpha1.ObjectReference{
						Name: "abc",
					},
				},
			}
			allErrs := validation.ValidateManifestExport(fld, export)
			Expect(allErrs).To(HaveLen(0))
		})

		It("should deny if required fields are missing", func() {
			export := &managedresource.Export{}
			allErrs := validation.ValidateManifestExport(fld, export)
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("a.key"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("a.jsonPath"),
			}))))
		})

		It("should deny if a invalid FromResource is defined", func() {
			export := &managedresource.Export{
				FromResource: &lsv1alpha1.TypedObjectReference{},
			}
			allErrs := validation.ValidateManifestExport(fld, export)
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("a.fromResource.apiVersion"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("a.fromResource.kind"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("a.fromResource.name"),
			}))))
		})

		It("should deny if a invalid fromObjectRef is defined", func() {
			export := &managedresource.Export{
				FromObjectReference: &managedresource.FromObjectReference{},
			}
			allErrs := validation.ValidateManifestExport(fld, export)
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("a.fromObjectRef.apiVersion"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("a.fromObjectRef.kind"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("a.fromObjectRef.jsonPath"),
			}))))
		})
	})

	Context("Deletion groups", func() {

		It("should accept valid deletion groups", func() {
			deletionGroups := []managedresource.DeletionGroupDefinition{
				{PredefinedResourceGroup: &managedresource.PredefinedResourceGroup{
					Type: managedresource.PredefinedResourceGroupNamespacedResources,
				}},
				{PredefinedResourceGroup: &managedresource.PredefinedResourceGroup{
					Type: managedresource.PredefinedResourceGroupClusterScopedResources,
				}},
				{PredefinedResourceGroup: &managedresource.PredefinedResourceGroup{
					Type: managedresource.PredefinedResourceGroupCRDs,
				}},
				{PredefinedResourceGroup: &managedresource.PredefinedResourceGroup{
					Type: managedresource.PredefinedResourceGroupEmpty,
				}},
				{CustomResourceGroup: &managedresource.CustomResourceGroup{
					Resources: []managedresource.ResourceType{
						{APIVersion: "landscaper.gardener.cloud/v1alpha1", Kind: "TestObject"},
						{APIVersion: "landscaper.gardener.cloud/v1alpha1", Kind: "ClusterTestObject"},
					},
				}},
			}
			allErrs := validation.ValidateDeletionGroups(fld, deletionGroups)
			Expect(allErrs).To(HaveLen(0))
		})

		It("should accept if no deletion groups are defined", func() {
			allErrs := validation.ValidateDeletionGroups(fld, nil)
			Expect(allErrs).To(HaveLen(0))

			allErrs = validation.ValidateDeletionGroups(fld, []managedresource.DeletionGroupDefinition{})
			Expect(allErrs).To(HaveLen(0))
		})

		It("should reject a deletion group that is neither predefined nor custom", func() {
			deletionGroups := []managedresource.DeletionGroupDefinition{
				{},
			}
			allErrs := validation.ValidateDeletionGroups(fld, deletionGroups)
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("a[0]"),
			}))))
		})

		It("should reject a deletion group that is predefined and custom", func() {
			deletionGroups := []managedresource.DeletionGroupDefinition{
				{
					PredefinedResourceGroup: &managedresource.PredefinedResourceGroup{},
					CustomResourceGroup:     &managedresource.CustomResourceGroup{},
				},
			}
			allErrs := validation.ValidateDeletionGroups(fld, deletionGroups)
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("a[0]"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("a[0].predefinedResourceGroup.type"),
			}))))
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("a[0].customResourceGroup.resources"),
			}))))
		})

		It("should reject a predefined deletion group with unsupported type", func() {
			deletionGroups := []managedresource.DeletionGroupDefinition{
				{PredefinedResourceGroup: &managedresource.PredefinedResourceGroup{
					Type: "test",
				}},
			}
			allErrs := validation.ValidateDeletionGroups(fld, deletionGroups)
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeNotSupported),
				"Field": Equal("a[0].predefinedResourceGroup.type"),
			}))))
		})

	})
})
