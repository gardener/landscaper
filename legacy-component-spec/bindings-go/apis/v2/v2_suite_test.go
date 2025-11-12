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

package v2_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "v2 Test Suite")
}

var _ = Describe("Helper", func() {

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
				Type:    v2.OCIImageType,
				ExtraIdentity: map[string]string{
					"all": "true",
				},
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
				Type:    v2.OCIImageType,
				ExtraIdentity: map[string]string{
					"all": "true",
				},
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

	Context("#IdentitySelector", func() {

		Context("json selector", func() {
			It("should select a resource by a single name selector", func() {
				res, err := comp.GetResourceByDefaultSelector(`{ "name": "image1"}`)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(HaveLen(1))
				Expect(res[0].Name).To(Equal("image1"))
			})
		})

		Context("go selector", func() {
			It("should select a resource by a single name selector", func() {
				res, err := comp.GetResourceByDefaultSelector(map[string]interface{}{
					"name": "image1",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(HaveLen(1))
				Expect(res[0].Name).To(Equal("image1"))
			})

			It("should select a resource by a regex selector", func() {
				res, err := comp.GetResourceByRegexSelector(map[string]interface{}{
					"name": ".*1",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(HaveLen(1))
				Expect(res[0].Name).To(Equal("image1"))
			})

			It("should select multiple resource by a regex selector", func() {
				res, err := comp.GetResourceByRegexSelector(map[string]interface{}{
					"name": "image.*",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(HaveLen(2))
			})

			It("should return no resource if a label does not match", func() {
				res, err := comp.GetResourceByDefaultSelector(map[string]interface{}{
					"name":    "image1",
					"nomatch": "fail",
				})
				Expect(err).To(Equal(v2.ErrNotFound))
				Expect(res).To(HaveLen(0))
			})

			It("should return multiple resources", func() {
				res, err := comp.GetResourceByDefaultSelector(map[string]interface{}{
					"all": "true",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(HaveLen(2))
			})
		})
	})

	It("should select a resource by a its type, name and version", func() {
		res, err := comp.GetExternalResource(v2.OCIImageType, "image1", "1.2.3")
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Name).To(Equal("image1"))
		Expect(res.Version).To(Equal("1.2.3"))
	})

	It("should select 2 resources by a their type", func() {
		res, err := comp.GetResourcesByType(v2.OCIImageType)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveLen(2))
	})

	It("should select no resources by a their type", func() {
		_, err := comp.GetResourcesByType(v2.GitType)
		Expect(err).To(HaveOccurred())
		Expect(err).To(Equal(v2.ErrNotFound))
	})

})
