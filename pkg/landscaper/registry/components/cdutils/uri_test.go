// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cdutils_test

import (
	"context"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/open-component-model/ocm/pkg/runtime"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/pkg/components/registries"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
	testutils "github.com/gardener/landscaper/test/utils"
)

var _ = Describe("URI", func() {
	var (
		ctx  context.Context
		octx ocm.Context

		registryAccess     model.RegistryAccess
		componentVersion   model.ComponentVersion
		repositorySpec     *types.UnstructuredTypedObject
		repositoryContext  = testutils.ExampleRepositoryContext()
		repositoryContexts = []*types.UnstructuredTypedObject{repositoryContext}
		cd                 types.ComponentDescriptor
		cd2                types.ComponentDescriptor
		resource1          types.Resource
		resource2          types.Resource
	)

	BeforeEach(func() {
		ctx = logging.NewContext(context.Background(), logging.Discard())
		octx = ocm.New(datacontext.MODE_EXTENDED)
		ctx = octx.BindTo(ctx)

		access := &cdv2.UnstructuredTypedObject{}
		err := access.UnmarshalJSON([]byte(`{"type":"localFilesystemBlob","fileName":"r1","mediaType":"example"}`))
		Expect(err).ToNot(HaveOccurred())
		resource1 = types.Resource{
			IdentityObjectMeta: cdv2.IdentityObjectMeta{
				Type:    "example",
				Name:    "r1",
				Version: "v1.0.0",
			},
			Relation: cdv2.LocalRelation,
			Access:   access,
		}
		err = access.UnmarshalJSON([]byte(`{"type":"localFilesystemBlob","fileName":"r2","mediaType":"example"}`))
		Expect(err).ToNot(HaveOccurred())
		resource2 = types.Resource{
			IdentityObjectMeta: cdv2.IdentityObjectMeta{
				Type:    "example",
				Name:    "r2",
				Version: "v0.0.0",
			},
			Relation: cdv2.ExternalRelation,
			Access:   access,
		}

		cd = types.ComponentDescriptor{
			Metadata: cdv2.Metadata{
				Version: "v2",
			},
			ComponentSpec: cdv2.ComponentSpec{
				ObjectMeta: cdv2.ObjectMeta{
					Name:    "example.com/comp",
					Version: "v1.0.0",
				},
				RepositoryContexts: repositoryContexts,
				Provider:           "external",
				ComponentReferences: []types.ComponentReference{
					{
						Name:          "comp1",
						ComponentName: "example.com/mycomp1",
						Version:       "v0.0.0",
					},
				},
				Resources: []types.Resource{resource1},
				Sources:   []types.Source{},
			},
		}

		cd2 = types.ComponentDescriptor{
			Metadata: cdv2.Metadata{
				Version: "v2",
			},
			ComponentSpec: cdv2.ComponentSpec{
				ObjectMeta: cdv2.ObjectMeta{
					Name:    "example.com/mycomp1",
					Version: "v0.0.0",
				},
				RepositoryContexts:  repositoryContexts,
				Provider:            "landscaper",
				Resources:           []types.Resource{resource2},
				Sources:             []types.Source{},
				ComponentReferences: []types.ComponentReference{},
			},
		}

		// Prepare in memory test repository
		memFs := memoryfs.New()

		// Write component descriptors to test repository
		file, err := memFs.Create("cd1-component-descriptor.yaml")
		Expect(err).ToNot(HaveOccurred())
		cd1data, err := runtime.DefaultYAMLEncoding.Marshal(cd)
		Expect(err).ToNot(HaveOccurred())
		_, err = file.Write(cd1data)
		Expect(err).ToNot(HaveOccurred())

		file, err = memFs.Create("cd2-component-descriptor.yaml")
		Expect(err).ToNot(HaveOccurred())
		cd2data, err := runtime.DefaultYAMLEncoding.Marshal(cd2)
		Expect(err).ToNot(HaveOccurred())
		_, err = file.Write(cd2data)
		Expect(err).ToNot(HaveOccurred())

		registryAccess, err = registries.GetFactory().NewRegistryAccess(ctx, memFs, nil, nil, nil,
			&config.LocalRegistryConfiguration{RootPath: "./"}, nil, nil)
		Expect(err).ToNot(HaveOccurred())

		repositorySpec = &cdv2.UnstructuredTypedObject{}
		err = repositorySpec.UnmarshalJSON([]byte(`{"type": "local", "filepath": "./"}`))
		Expect(err).ToNot(HaveOccurred())

		componentVersion, err = registryAccess.GetComponentVersion(ctx, &lsv1alpha1.ComponentDescriptorReference{
			RepositoryContext: repositorySpec,
			ComponentName:     cd.GetName(),
			Version:           cd.GetVersion(),
		})
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(octx.Finalize()).To(Succeed())
	})

	It("should resolve a direct local resource", func() {
		uri, err := cdutils.ParseURI("cd://resources/r1")
		Expect(err).ToNot(HaveOccurred())
		repoContext := componentVersion.GetRepositoryContext()
		kind, res, err := uri.Get(componentVersion, repoContext)
		Expect(err).ToNot(HaveOccurred())
		Expect(kind).To(Equal(lsv1alpha1.ResourceKind))
		resource, ok := res.(model.Resource)
		Expect(ok).To(BeTrue())
		resourceEntry, err := resource.GetResource()
		Expect(err).NotTo(HaveOccurred())
		// ignore the raw part because order of a marshaled map is unpredictable
		resourceEntry.Access.Raw = []byte{}
		resource1.Access.Raw = []byte{}
		Expect(resourceEntry).To(Equal(&resource1))
	})

	It("should return an error if a resource is unknown", func() {
		uri, err := cdutils.ParseURI("cd://resources/r3")
		Expect(err).ToNot(HaveOccurred())
		//repoContext, err := componentVersion.GetRepositoryContext()
		//Expect(err).NotTo(HaveOccurred())
		_, _, err = uri.Get(componentVersion, repositorySpec)
		Expect(err).To(HaveOccurred())
	})

	It("should return an error if a keyword is unknown", func() {
		uri, err := cdutils.ParseURI("cd://fail/r1")
		Expect(err).ToNot(HaveOccurred())
		//repoContext, err := componentVersion.GetRepositoryContext()
		//Expect(err).NotTo(HaveOccurred())
		_, _, err = uri.Get(componentVersion, repositorySpec)
		Expect(err).To(HaveOccurred())
	})

	It("should resolve a component reference", func() {
		uri, err := cdutils.ParseURI("cd://componentReferences/comp1")
		Expect(err).ToNot(HaveOccurred())
		//repoContext, err := componentVersion.GetRepositoryContext()
		//Expect(err).NotTo(HaveOccurred())
		kind, res, err := uri.Get(componentVersion, repositorySpec)
		Expect(err).ToNot(HaveOccurred())
		Expect(kind).To(Equal(lsv1alpha1.ComponentResourceKind))
		component, ok := res.(model.ComponentVersion)
		Expect(ok).To(BeTrue())
		componentDescriptor := component.GetComponentDescriptor()
		// ignore the raw part because order of a marshaled map is unpredictable
		componentDescriptor.RepositoryContexts[0].Raw = []byte{}
		componentDescriptor.Resources[0].Access.Raw = []byte{}
		cd2.RepositoryContexts[0].Raw = []byte{}
		cd2.Resources[0].Access.Raw = []byte{}
		Expect(componentDescriptor).To(Equal(&cd2))
	})

	It("should resolve a resource in a component reference", func() {
		uri, err := cdutils.ParseURI("cd://componentReferences/comp1/resources/r2")
		Expect(err).ToNot(HaveOccurred())
		//repoContext, err := componentVersion.GetRepositoryContext()
		//Expect(err).NotTo(HaveOccurred())
		kind, res, err := uri.Get(componentVersion, repositorySpec)
		Expect(err).ToNot(HaveOccurred())
		Expect(kind).To(Equal(lsv1alpha1.ResourceKind))
		resource, ok := res.(model.Resource)
		Expect(ok).To(BeTrue())
		resourceEntry, err := resource.GetResource()
		Expect(err).NotTo(HaveOccurred())
		// ignore the raw part because order of a marshaled map is unpredictable
		resourceEntry.Access.Raw = []byte{}
		resource1.Access.Raw = []byte{}
		Expect(resourceEntry).To(Equal(&resource2))
	})
})
