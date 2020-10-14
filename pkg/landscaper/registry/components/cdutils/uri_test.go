// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cdutils_test

import (
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
)

var _ = Describe("URI", func() {
	var (
		repoCtx = []cdv2.RepositoryContext{
			{Type: cdv2.OCIRegistryType, BaseURL: "example.com"},
		}
		cd            cdutils.ResolvedComponentDescriptor
		testResources = map[string]cdv2.Resource{
			"r1": {
				ObjectMeta: cdv2.ObjectMeta{
					Name:    "r1",
					Version: "1.5.5",
				},
			},
			"r2": {
				ObjectMeta: cdv2.ObjectMeta{
					Name:    "r2",
					Version: "1.5.0",
				},
			},
		}
	)

	BeforeEach(func() {
		cd = cdutils.ResolvedComponentDescriptor{}
		cd.ObjectMeta = cdv2.ObjectMeta{
			Name:    "comp",
			Version: "1.0.0",
		}
		spec := cdutils.ResolvedComponentSpec{}
		spec.Provider = cdv2.ExternalProvider
		spec.RepositoryContexts = repoCtx
		cd.ResolvedComponentSpec = spec
	})

	It("should resolve a direct local resource", func() {
		cd.LocalResources = testResources
		uri, err := cdutils.ParseURI("cd://localResources/r1")
		Expect(err).ToNot(HaveOccurred())
		kind, res, err := uri.Get(cd)
		Expect(err).ToNot(HaveOccurred())
		Expect(kind).To(Equal(lsv1alpha1.LocalResourceKind))
		Expect(res).To(Equal(testResources["r1"]))
	})

	It("should return an error if a keyword is unknown or wrong", func() {
		cd.LocalResources = testResources
		uri, err := cdutils.ParseURI("cd://localResource/r1")
		Expect(err).ToNot(HaveOccurred())
		_, _, err = uri.Get(cd)
		Expect(err).To(HaveOccurred())

		uri, err = cdutils.ParseURI("cd://fail/r1")
		Expect(err).ToNot(HaveOccurred())
		_, _, err = uri.Get(cd)
		Expect(err).To(HaveOccurred())
	})

	It("should resolve a component reference", func() {
		comp1 := cdutils.ResolvedComponentDescriptor{
			ResolvedComponentSpec: cdutils.ResolvedComponentSpec{
				ObjectMeta: cdv2.ObjectMeta{
					Name: "comp1",
				},
			},
		}
		cd.ComponentReferences = map[string]cdutils.ResolvedComponentDescriptor{
			"comp1": comp1,
		}
		uri, err := cdutils.ParseURI("cd://componentReferences/comp1")
		Expect(err).ToNot(HaveOccurred())
		kind, res, err := uri.Get(cd)
		Expect(err).ToNot(HaveOccurred())
		Expect(kind).To(Equal(lsv1alpha1.ComponentResourceKind))
		Expect(res).To(Equal(comp1))
	})

	It("should resolve a resource in a component reference", func() {
		comp1 := cdutils.ResolvedComponentDescriptor{
			ResolvedComponentSpec: cdutils.ResolvedComponentSpec{
				ObjectMeta: cdv2.ObjectMeta{
					Name: "comp1",
				},
				LocalResources: testResources,
			},
		}
		cd.ComponentReferences = map[string]cdutils.ResolvedComponentDescriptor{
			"comp1": comp1,
		}
		uri, err := cdutils.ParseURI("cd://componentReferences/comp1/localResources/r1")
		Expect(err).ToNot(HaveOccurred())
		kind, res, err := uri.Get(cd)
		Expect(err).ToNot(HaveOccurred())
		Expect(kind).To(Equal(lsv1alpha1.LocalResourceKind))
		Expect(res).To(Equal(testResources["r1"]))
	})
})
