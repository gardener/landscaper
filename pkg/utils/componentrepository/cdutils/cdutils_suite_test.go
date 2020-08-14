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

package cdutils_test

import (
	"testing"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/pkg/utils/componentrepository/cdutils"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "landscaper component descriptor")
}

var _ = Describe("mapped component descriptor", func() {

	Context("#MappedComponentDescriptor", func() {

		testResources := []cdv2.Resource{
			{
				ObjectMeta: cdv2.ObjectMeta{
					Name:    "r1",
					Version: "1.5.5",
				},
			},
			{
				ObjectMeta: cdv2.ObjectMeta{
					Name:    "r2",
					Version: "1.5.0",
				},
			},
		}

		It("should map default metadata", func() {
			cd := cdv2.ComponentDescriptor{}
			cd.ObjectMeta = cdv2.ObjectMeta{
				Name:    "comp",
				Version: "1.0.0",
			}
			cd.Provider = cdv2.ExternalProvider
			cd.RepositoryContexts = []cdv2.RepositoryContext{
				{
					BaseURL: "http://example.com",
				},
			}

			mcd := cdutils.ConvertFromComponentDescriptor(cd)
			Expect(mcd.Name).To(Equal("comp"))
			Expect(mcd.Version).To(Equal("1.0.0"))
			Expect(mcd.Provider).To(Equal(cdv2.ExternalProvider))
			Expect(mcd.RepositoryContexts).To(ConsistOf(cd.RepositoryContexts[0]))
		})

		It("should convert a list sources to a map by the resource name", func() {
			cd := cdv2.ComponentDescriptor{}
			cd.Sources = testResources

			mcd := cdutils.ConvertFromComponentDescriptor(cd)
			Expect(mcd.Sources).To(HaveKeyWithValue("r1", testResources[0]))
			Expect(mcd.Sources).To(HaveKeyWithValue("r2", testResources[1]))
		})

		It("should convert a list local resources to a map by the resource name", func() {
			cd := cdv2.ComponentDescriptor{}
			cd.LocalResources = testResources

			mcd := cdutils.ConvertFromComponentDescriptor(cd)
			Expect(mcd.LocalResources).To(HaveKeyWithValue("r1", testResources[0]))
			Expect(mcd.LocalResources).To(HaveKeyWithValue("r2", testResources[1]))
		})

		It("should convert a list external resources to a map by the resource name", func() {
			cd := cdv2.ComponentDescriptor{}
			cd.ExternalResources = testResources

			mcd := cdutils.ConvertFromComponentDescriptor(cd)
			Expect(mcd.ExternalResources).To(HaveKeyWithValue("r1", testResources[0]))
			Expect(mcd.ExternalResources).To(HaveKeyWithValue("r2", testResources[1]))
		})

		It("should convert a list component references to a map by the reference's name", func() {
			cd := cdv2.ComponentDescriptor{}
			cd.ComponentReferences = []cdv2.ObjectMeta{
				{
					Name:    "ref1",
					Version: "1.0.0",
				},
				{
					Name:    "ref2",
					Version: "1.0.1",
				},
			}

			mcd := cdutils.ConvertFromComponentDescriptor(cd)
			Expect(mcd.ComponentReferences).To(HaveKeyWithValue("ref1", cd.ComponentReferences[0]))
			Expect(mcd.ComponentReferences).To(HaveKeyWithValue("ref2", cd.ComponentReferences[1]))
		})
	})

})
