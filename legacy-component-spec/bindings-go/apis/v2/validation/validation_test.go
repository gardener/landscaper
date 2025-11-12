// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package validation

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"

	v2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "V2 Test Suite")
}

var _ = Describe("Validation", func() {

	var (
		comp *v2.ComponentDescriptor

		ociImage1    *v2.Resource
		ociRegistry1 *v2.OCIRegistryAccess
		ociImage2    *v2.Resource
		ociRegistry2 *v2.OCIRegistryAccess
	)

	BeforeEach(func() {
		ociRegistry1 = &v2.OCIRegistryAccess{
			ObjectType: v2.ObjectType{
				Type: v2.OCIRegistryType,
			},
			ImageReference: "docker/image1:1.2.3",
		}

		unstrucOCIRegistry1, err := v2.NewUnstructured(ociRegistry1)
		Expect(err).ToNot(HaveOccurred())

		ociImage1 = &v2.Resource{
			IdentityObjectMeta: v2.IdentityObjectMeta{
				Name:    "image1",
				Version: "1.2.3",
			},
			Relation: v2.ExternalRelation,
			Access:   &unstrucOCIRegistry1,
		}
		ociRegistry2 = &v2.OCIRegistryAccess{
			ObjectType: v2.ObjectType{
				Type: v2.OCIRegistryType,
			},
			ImageReference: "docker/image1:1.2.3",
		}
		unstrucOCIRegistry2, err := v2.NewUnstructured(ociRegistry2)
		Expect(err).ToNot(HaveOccurred())
		ociImage2 = &v2.Resource{
			IdentityObjectMeta: v2.IdentityObjectMeta{
				Name:    "image2",
				Version: "1.2.3",
			},
			Relation: v2.ExternalRelation,
			Access:   &unstrucOCIRegistry2,
		}

		comp = &v2.ComponentDescriptor{
			Metadata: v2.Metadata{
				Version: v2.SchemaVersion,
			},
			ComponentSpec: v2.ComponentSpec{
				ObjectMeta: v2.ObjectMeta{
					Name:    "my-comp",
					Version: "1.2.3",
				},
				Provider:            "external",
				RepositoryContexts:  nil,
				Sources:             nil,
				ComponentReferences: nil,
				Resources:           []v2.Resource{*ociImage1, *ociImage2},
			},
		}
	})

	Context("#Metadata", func() {

		It("should forbid if the component schemaVersion is missing", func() {
			comp := v2.ComponentDescriptor{
				Metadata: v2.Metadata{},
			}

			errList := validate(nil, &comp)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("meta.schemaVersion"),
			}))))
		})

		It("should pass if the component schemaVersion is defined", func() {
			errList := validate(nil, comp)
			Expect(errList).ToNot(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("meta.schemaVersion"),
			}))))
		})

	})

	Context("#Provider", func() {
		It("should pass if a component's provider is a non-empty string", func() {
			comp.Provider = "custom"
			errList := validate(nil, comp)
			Expect(errList).ToNot(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("component.provider"),
			}))))
		})
	})

	Context("#ObjectMeta", func() {
		It("should forbid invalid component name as specified in json schema", func() {
			comp.Name = "http://example.org/org/name"
			err := v2.DefaultComponent(comp)
			Expect(err).ToNot(HaveOccurred())
			errs := Validate(comp)
			Expect(errs).To(HaveOccurred())
			Expect(errs.Error()).To(ContainSubstring("component.name: Does not match pattern"))
		})

		It("should forbid if the component's version is missing", func() {
			comp := v2.ComponentDescriptor{}
			errList := validate(nil, &comp)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("component.name"),
			}))))
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("component.version"),
			}))))
		})

		It("should forbid if the component's name is missing", func() {
			comp := v2.ComponentDescriptor{}
			errList := validate(nil, &comp)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("component.name"),
			}))))
		})

	})

	Context("#Sources", func() {
		It("should forbid if a duplicated component's source is defined", func() {
			comp.Sources = []v2.Source{
				{
					IdentityObjectMeta: v2.IdentityObjectMeta{
						Name: "a",
					},
					Access: v2.NewEmptyUnstructured("custom"),
				},
				{
					IdentityObjectMeta: v2.IdentityObjectMeta{
						Name: "a",
					},
					Access: v2.NewEmptyUnstructured("custom"),
				},
			}
			errList := validate(nil, comp)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeDuplicate),
				"Field": Equal("component.sources[1]"),
			}))))
		})
	})

	Context("#ComponentReferences", func() {
		It("should pass if a reference is set", func() {
			comp.ComponentReferences = []v2.ComponentReference{
				{
					Name:          "test",
					ComponentName: "test",
					Version:       "1.2.3",
				},
			}
			errList := validate(nil, comp)
			Expect(errList).ToNot(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("component.componentReferences[0].name"),
			}))))
			Expect(errList).ToNot(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("component.componentReferences[0].version"),
			}))))
		})

		It("should forbid if a reference's name is missing", func() {
			comp.ComponentReferences = []v2.ComponentReference{
				{
					ComponentName: "test",
					Version:       "1.2.3",
				},
			}
			errList := validate(nil, comp)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("component.componentReferences[0].name"),
			}))))
		})

		It("should forbid if a reference's component name is missing", func() {
			comp.ComponentReferences = []v2.ComponentReference{
				{
					Name:    "test",
					Version: "1.2.3",
				},
			}
			errList := validate(nil, comp)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("component.componentReferences[0].componentName"),
			}))))
		})

		It("should forbid if a reference's version is missing", func() {
			comp.ComponentReferences = []v2.ComponentReference{
				{
					ComponentName: "test",
					Name:          "test",
				},
			}
			errList := validate(nil, comp)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("component.componentReferences[0].version"),
			}))))
		})

	})

	Context("#Resources", func() {
		It("should forbid if a local resource's version differs from the version of the parent", func() {
			comp.Resources = []v2.Resource{
				{
					IdentityObjectMeta: v2.IdentityObjectMeta{
						Name:    "locRes",
						Version: "0.0.1",
					},
					Relation: v2.LocalRelation,
					Access:   v2.NewEmptyUnstructured(v2.OCIImageType),
				},
			}
			errList := validate(nil, comp)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("component.resources[0].version"),
			}))))
		})

		It("should forbid if a resource name contains invalid characters", func() {
			comp.Resources = []v2.Resource{
				{
					IdentityObjectMeta: v2.IdentityObjectMeta{
						Name: "test$",
					},
				},
				{
					IdentityObjectMeta: v2.IdentityObjectMeta{
						Name: "test🙅",
					},
				},
			}
			errList := validate(nil, comp)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("component.resources[0].name"),
			}))))
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("component.resources[1].name"),
			}))))
		})

		It("should forbid if a duplicated local resource is defined", func() {
			comp.Resources = []v2.Resource{
				{
					IdentityObjectMeta: v2.IdentityObjectMeta{
						Name: "test",
					},
				},
				{
					IdentityObjectMeta: v2.IdentityObjectMeta{
						Name: "test",
					},
				},
			}
			errList := validate(nil, comp)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeDuplicate),
				"Field": Equal("component.resources[1]"),
			}))))
		})

		It("should forbid if a duplicated resource with additional identity labels is defined", func() {
			comp.Resources = []v2.Resource{
				{
					IdentityObjectMeta: v2.IdentityObjectMeta{
						Name: "test",
						ExtraIdentity: v2.Identity{
							"my-id": "some-id",
						},
					},
				},
				{
					IdentityObjectMeta: v2.IdentityObjectMeta{
						Name: "test",
						ExtraIdentity: v2.Identity{
							"my-id": "some-id",
						},
					},
				},
			}
			errList := validate(nil, comp)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeDuplicate),
				"Field": Equal("component.resources[1]"),
			}))))
		})

		It("should pass if a duplicated resource has the same name but with different additional identity labels", func() {
			comp.Resources = []v2.Resource{
				{
					IdentityObjectMeta: v2.IdentityObjectMeta{
						Name: "test",
						ExtraIdentity: v2.Identity{
							"my-id": "some-id",
						},
					},
				},
				{
					IdentityObjectMeta: v2.IdentityObjectMeta{
						Name: "test",
					},
				},
			}
			errList := validate(nil, comp)
			Expect(errList).ToNot(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeDuplicate),
				"Field": Equal("component.resources[1]"),
			}))))
			Expect(errList).ToNot(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeDuplicate),
				"Field": Equal("component.resources[0]"),
			}))))
		})
	})

	Context("#labels", func() {

		It("should forbid if labels are defined multiple times in the same context", func() {
			comp.ComponentReferences = []v2.ComponentReference{
				{
					ComponentName: "test",
					Name:          "test",
					Version:       "1.2.3",
					Labels: []v2.Label{
						{
							Name:  "l1",
							Value: []byte{},
						},
						{
							Name:  "l1",
							Value: []byte{},
						},
					},
				},
			}

			errList := validate(nil, comp)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeDuplicate),
				"Field": Equal("component.componentReferences[0].labels[1]"),
			}))))
		})

		It("should pass if labels are defined multiple times in the same context with differnet names", func() {
			comp.ComponentReferences = []v2.ComponentReference{
				{
					ComponentName: "test",
					Name:          "test",
					Version:       "1.2.3",
					Labels: []v2.Label{
						{
							Name:  "l1",
							Value: []byte{},
						},
						{
							Name:  "l2",
							Value: []byte{},
						},
					},
				},
			}

			errList := validate(nil, comp)
			Expect(errList).ToNot(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeDuplicate),
				"Field": Equal("component.componentReferences[0].labels[1]"),
			}))))
		})
	})

	Context("#Identity", func() {
		It("should pass valid identity labels", func() {
			identity := v2.Identity{
				"my-l1": "test",
				"my-l2": "test",
			}
			errList := ValidateIdentity(field.NewPath("identity"), identity)
			Expect(errList).To(HaveLen(0))
		})

		It("should forbid if a identity label define the name", func() {
			identity := v2.Identity{
				"name": "test",
			}
			errList := ValidateIdentity(field.NewPath("identity"), identity)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeForbidden),
				"Field": Equal("identity[name]"),
			}))))
		})

		It("should forbid if a identity label defines a key with invalid characters", func() {
			identity := v2.Identity{
				"my-l1!": "test",
			}
			errList := ValidateIdentity(field.NewPath("identity"), identity)
			Expect(errList).ToNot(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeForbidden),
				"Field": Equal("identity[my-l1!]"),
			}))))
		})
	})
})
