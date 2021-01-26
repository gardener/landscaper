// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cdutils_test

import (
	"context"
	"errors"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
)

var _ = Describe("URI", func() {
	var (
		compRefResolver = func(_ context.Context, meta cdv2.ComponentReference) (cdv2.ComponentDescriptor, error) {
			return cdv2.ComponentDescriptor{}, errors.New("NotFound")
		}
		repoCtx = []cdv2.RepositoryContext{
			{Type: cdv2.OCIRegistryType, BaseURL: "example.com"},
		}
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
		kind, res, err := uri.Get(cd, compRefResolver)
		Expect(err).ToNot(HaveOccurred())
		Expect(kind).To(Equal(lsv1alpha1.ResourceKind))
		Expect(res).To(Equal(testResources[0]))
	})

	It("should return an error if a keyword is unknown or wrong", func() {
		cd.Resources = testResources
		uri, err := cdutils.ParseURI("cd://resources/r3")
		Expect(err).ToNot(HaveOccurred())
		_, _, err = uri.Get(cd, compRefResolver)
		Expect(err).To(HaveOccurred())

		uri, err = cdutils.ParseURI("cd://fail/r1")
		Expect(err).ToNot(HaveOccurred())
		_, _, err = uri.Get(cd, compRefResolver)
		Expect(err).To(HaveOccurred())
	})

	It("should resolve a component reference", func() {
		comp1 := cdv2.ComponentDescriptor{
			ComponentSpec: cdv2.ComponentSpec{
				ObjectMeta: cdv2.ObjectMeta{
					Name: "comp1",
				},
			},
		}
		cd.ComponentReferences = []cdv2.ComponentReference{
			{
				Name: "comp1",
			},
		}
		compRefResolver = func(_ context.Context, meta cdv2.ComponentReference) (cdv2.ComponentDescriptor, error) {
			return comp1, nil
		}
		uri, err := cdutils.ParseURI("cd://componentReferences/comp1")
		Expect(err).ToNot(HaveOccurred())
		kind, res, err := uri.Get(cd, compRefResolver)
		Expect(err).ToNot(HaveOccurred())
		Expect(kind).To(Equal(lsv1alpha1.ComponentResourceKind))
		Expect(res).To(Equal(&comp1))
	})

	It("should resolve a resource in a component reference", func() {
		comp1 := cdv2.ComponentDescriptor{
			ComponentSpec: cdv2.ComponentSpec{
				ObjectMeta: cdv2.ObjectMeta{
					Name: "comp1",
				},
				Resources: testResources,
			},
		}
		cd.ComponentReferences = []cdv2.ComponentReference{
			{
				Name: "comp1",
			},
		}
		compRefResolver = func(_ context.Context, meta cdv2.ComponentReference) (cdv2.ComponentDescriptor, error) {
			return comp1, nil
		}
		uri, err := cdutils.ParseURI("cd://componentReferences/comp1/resources/r1")
		Expect(err).ToNot(HaveOccurred())
		kind, res, err := uri.Get(cd, compRefResolver)
		Expect(err).ToNot(HaveOccurred())
		Expect(kind).To(Equal(lsv1alpha1.ResourceKind))
		Expect(res).To(Equal(testResources[0]))
	})
})
