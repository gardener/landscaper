// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cdutils_test

import (
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	testutils "github.com/gardener/landscaper/test/utils"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
)

var _ = Describe("URI", func() {
	var (
		repoCtx       = []*cdv2.UnstructuredTypedObject{testutils.ExampleRepositoryContext()}
		cd            *cdv2.ComponentDescriptor
		testResources = []cdv2.Resource{
			{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "r1",
					Version: "1.5.5",
				},
				Relation: cdv2.LocalRelation,
			},
			{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "r2",
					Version: "1.5.0",
				},
				Relation: cdv2.ExternalRelation,
			},
		}
	)

	BeforeEach(func() {
		cd = &cdv2.ComponentDescriptor{}
		cd.ObjectMeta = cdv2.ObjectMeta{
			Name:    "comp",
			Version: "1.0.0",
		}
		spec := cdv2.ComponentSpec{}
		spec.Provider = cdv2.ExternalProvider
		spec.RepositoryContexts = repoCtx
		cd.ComponentSpec = spec
	})

	It("should resolve a direct local resource", func() {
		cd.Resources = testResources
		uri, err := cdutils.ParseURI("cd://resources/r1")
		Expect(err).ToNot(HaveOccurred())
		kind, res, err := uri.Get(cd, nil, cd.GetEffectiveRepositoryContext())
		Expect(err).ToNot(HaveOccurred())
		Expect(kind).To(Equal(lsv1alpha1.ResourceKind))
		Expect(res).To(Equal(testResources[0]))
	})

	It("should return an error if a keyword is unknown or wrong", func() {
		cd.Resources = testResources
		uri, err := cdutils.ParseURI("cd://resources/r3")
		Expect(err).ToNot(HaveOccurred())
		_, _, err = uri.Get(cd, nil, cd.GetEffectiveRepositoryContext())
		Expect(err).To(HaveOccurred())

		uri, err = cdutils.ParseURI("cd://fail/r1")
		Expect(err).ToNot(HaveOccurred())
		_, _, err = uri.Get(cd, nil, cd.GetEffectiveRepositoryContext())
		Expect(err).To(HaveOccurred())
	})

	It("should resolve a component reference", func() {
		comp1 := cdv2.ComponentDescriptor{
			ComponentSpec: cdv2.ComponentSpec{
				ObjectMeta: cdv2.ObjectMeta{
					Name:    "my-comp1",
					Version: "v0.0.0",
				},
				RepositoryContexts: []*cdv2.UnstructuredTypedObject{testutils.ExampleRepositoryContext()},
			},
		}
		cd.ComponentReferences = []cdv2.ComponentReference{
			{
				Name:          "comp1",
				ComponentName: "my-comp1",
				Version:       "v0.0.0",
			},
		}

		compResolver, err := ctf.NewListResolver(&cdv2.ComponentDescriptorList{
			Metadata: cdv2.Metadata{},
			Components: []cdv2.ComponentDescriptor{
				comp1,
			},
		})
		Expect(err).ToNot(HaveOccurred())
		uri, err := cdutils.ParseURI("cd://componentReferences/comp1")
		Expect(err).ToNot(HaveOccurred())
		kind, res, err := uri.Get(cd, compResolver, cd.GetEffectiveRepositoryContext())
		Expect(err).ToNot(HaveOccurred())
		Expect(kind).To(Equal(lsv1alpha1.ComponentResourceKind))
		Expect(res).To(Equal(&comp1))
	})

	It("should resolve a resource in a component reference", func() {
		comp1 := cdv2.ComponentDescriptor{
			ComponentSpec: cdv2.ComponentSpec{
				ObjectMeta: cdv2.ObjectMeta{
					Name:    "my-comp1",
					Version: "v0.0.0",
				},
				RepositoryContexts: []*cdv2.UnstructuredTypedObject{testutils.ExampleRepositoryContext()},
				Resources:          testResources,
			},
		}
		cd.ComponentReferences = []cdv2.ComponentReference{
			{
				Name:          "comp1",
				ComponentName: "my-comp1",
				Version:       "v0.0.0",
			},
		}
		compResolver, err := ctf.NewListResolver(&cdv2.ComponentDescriptorList{
			Metadata: cdv2.Metadata{},
			Components: []cdv2.ComponentDescriptor{
				comp1,
			},
		})
		Expect(err).ToNot(HaveOccurred())
		uri, err := cdutils.ParseURI("cd://componentReferences/comp1/resources/r1")
		Expect(err).ToNot(HaveOccurred())
		kind, res, err := uri.Get(cd, compResolver, cd.GetEffectiveRepositoryContext())
		Expect(err).ToNot(HaveOccurred())
		Expect(kind).To(Equal(lsv1alpha1.ResourceKind))
		Expect(res).To(Equal(testResources[0]))
	})
})
