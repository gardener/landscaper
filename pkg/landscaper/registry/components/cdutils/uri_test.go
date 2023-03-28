// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cdutils_test

import (
	"context"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
	componentstestutils "github.com/gardener/landscaper/pkg/components/testutils"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
	testutils "github.com/gardener/landscaper/test/utils"
)

var _ = Describe("URI", func() {
	var (
		registryAccess     model.RegistryAccess
		componentVersion   model.ComponentVersion
		repositoryContext  = testutils.ExampleRepositoryContext()
		repositoryContexts = []*cdv2.UnstructuredTypedObject{repositoryContext}

		resource1 = cdv2.Resource{
			IdentityObjectMeta: cdv2.IdentityObjectMeta{
				Name:    "r1",
				Version: "1.5.5",
			},
			Relation: cdv2.LocalRelation,
		}

		resource2 = cdv2.Resource{
			IdentityObjectMeta: cdv2.IdentityObjectMeta{
				Name:    "r2",
				Version: "1.5.0",
			},
			Relation: cdv2.ExternalRelation,
		}

		cd = cdv2.ComponentDescriptor{
			ComponentSpec: cdv2.ComponentSpec{
				ObjectMeta: cdv2.ObjectMeta{
					Name:    "comp",
					Version: "1.0.0",
				},
				RepositoryContexts: repositoryContexts,
				Provider:           cdv2.ExternalProvider,
				ComponentReferences: []cdv2.ComponentReference{
					{
						Name:          "comp1",
						ComponentName: "my-comp1",
						Version:       "v0.0.0",
					},
				},
				Resources: []cdv2.Resource{resource1},
			},
		}

		cd2 = cdv2.ComponentDescriptor{
			ComponentSpec: cdv2.ComponentSpec{
				ObjectMeta: cdv2.ObjectMeta{
					Name:    "my-comp1",
					Version: "v0.0.0",
				},
				RepositoryContexts: repositoryContexts,
				Resources:          []cdv2.Resource{resource2},
			},
		}
	)

	BeforeEach(func() {
		ctx := context.Background()

		registryAccess = componentstestutils.NewTestRegistryAccess(cd, cd2)

		var err error
		componentVersion, err = registryAccess.GetComponentVersion(ctx, &lsv1alpha1.ComponentDescriptorReference{
			RepositoryContext: repositoryContext,
			ComponentName:     cd.GetName(),
			Version:           cd.GetVersion(),
		})
		Expect(err).ToNot(HaveOccurred())
	})

	It("should resolve a direct local resource", func() {
		uri, err := cdutils.ParseURI("cd://resources/r1")
		Expect(err).ToNot(HaveOccurred())
		kind, res, err := uri.Get(componentVersion, componentVersion.GetRepositoryContext())
		Expect(err).ToNot(HaveOccurred())
		Expect(kind).To(Equal(lsv1alpha1.ResourceKind))
		resource, ok := res.(model.Resource)
		Expect(ok).To(BeTrue())
		Expect(resource.GetResource()).To(Equal(&resource1))
	})

	It("should return an error if a resource is unknown", func() {
		uri, err := cdutils.ParseURI("cd://resources/r3")
		Expect(err).ToNot(HaveOccurred())
		_, _, err = uri.Get(componentVersion, componentVersion.GetRepositoryContext())
		Expect(err).To(HaveOccurred())
	})

	It("should return an error if a keyword is unknown", func() {
		uri, err := cdutils.ParseURI("cd://fail/r1")
		Expect(err).ToNot(HaveOccurred())
		_, _, err = uri.Get(componentVersion, componentVersion.GetRepositoryContext())
		Expect(err).To(HaveOccurred())
	})

	It("should resolve a component reference", func() {
		uri, err := cdutils.ParseURI("cd://componentReferences/comp1")
		Expect(err).ToNot(HaveOccurred())
		kind, res, err := uri.Get(componentVersion, componentVersion.GetRepositoryContext())
		Expect(err).ToNot(HaveOccurred())
		Expect(kind).To(Equal(lsv1alpha1.ComponentResourceKind))
		component, ok := res.(model.ComponentVersion)
		Expect(ok).To(BeTrue())
		Expect(component.GetComponentDescriptor()).To(Equal(&cd2))
	})

	It("should resolve a resource in a component reference", func() {
		uri, err := cdutils.ParseURI("cd://componentReferences/comp1/resources/r2")
		Expect(err).ToNot(HaveOccurred())
		kind, res, err := uri.Get(componentVersion, componentVersion.GetRepositoryContext())
		Expect(err).ToNot(HaveOccurred())
		Expect(kind).To(Equal(lsv1alpha1.ResourceKind))
		resource, ok := res.(model.Resource)
		Expect(ok).To(BeTrue())
		Expect(resource.GetResource()).To(Equal(&resource2))
	})
})
