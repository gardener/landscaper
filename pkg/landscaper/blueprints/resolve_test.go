// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints_test

import (
	"context"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/apis/config/v1alpha1"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/mediatype"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/components/cnudie/componentresolvers"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/components/registries"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints/bputils"
	"github.com/gardener/landscaper/pkg/utils/blueprints"
	testutils "github.com/gardener/landscaper/test/utils"
)

var _ = Describe("Resolve", func() {

	var (
		ctx                context.Context
		octx               ocm.Context
		defaultStoreConfig config.BlueprintStore
	)

	BeforeEach(func() {
		ctx = logging.NewContext(context.Background(), logging.Discard())
		octx = ocm.New(datacontext.MODE_EXTENDED)
		ctx = octx.BindTo(ctx)

		cs := v1alpha1.BlueprintStore{}
		v1alpha1.SetDefaults_BlueprintStore(&cs)
		Expect(v1alpha1.Convert_v1alpha1_BlueprintStore_To_config_BlueprintStore(&cs, &defaultStoreConfig, nil)).To(Succeed())
	})
	AfterEach(func() {
		Expect(octx.Finalize()).To(Succeed())
	})

	// TODO: remove with component-cli
	Context("ResolveBlueprintFromBlobResolver", func() {
		It("should resolve a blueprint from a blobresolver", func() {
			memFs := memoryfs.New()
			err := bputils.NewBuilder().Blueprint(&lsv1alpha1.Blueprint{
				Annotations: map[string]string{
					"test": "val",
				},
			}).BuildResourceToFs(memFs, "blobs/bp.tar", false)
			Expect(err).ToNot(HaveOccurred())

			blobResolver := componentresolvers.NewLocalFilesystemBlobResolver(memFs)

			localFSAccess, err := componentresolvers.NewLocalFilesystemBlobAccess("bp.tar", mediatype.BlueprintArtifactsLayerMediaTypeV1)

			Expect(err).ToNot(HaveOccurred())

			repositoryContext := testutils.ExampleRepositoryContext()
			repositoryContexts := []*types.UnstructuredTypedObject{repositoryContext}

			cd := types.ComponentDescriptor{}
			cd.Metadata.Version = "v2"
			cd.Name = "example.com/a"
			cd.Version = "0.0.1"
			cd.Provider = "landscaper"
			cd.RepositoryContexts = repositoryContexts
			cd.Sources = []types.Source{}
			cd.Resources = append(cd.Resources, types.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-bp",
					Version: "1.2.3",
					Type:    mediatype.BlueprintType,
				},
				Relation: cdv2.ExternalRelation,
				Access:   &localFSAccess,
			})
			cd.ComponentReferences = []cdv2.ComponentReference{}

			registryAccess, err := registries.GetFactory().NewRegistryAccess(ctx, memFs, nil, nil, nil,
				&config.LocalRegistryConfiguration{RootPath: "./blobs"}, nil, &cd, blobResolver)
			Expect(err).ToNot(HaveOccurred())

			componentVersion, err := registryAccess.GetComponentVersion(ctx, &lsv1alpha1.ComponentDescriptorReference{
				RepositoryContext: repositoryContext,
				ComponentName:     cd.GetName(),
				Version:           cd.GetVersion(),
			})
			Expect(err).NotTo(HaveOccurred())

			resource, err := componentVersion.GetResource("my-bp", nil)
			Expect(err).NotTo(HaveOccurred())
			content, err := resource.GetTypedContent(ctx)
			Expect(err).ToNot(HaveOccurred())
			bp, ok := content.Resource.(*blueprints.Blueprint)
			Expect(ok).To(BeTrue())
			Expect(bp.Info.Annotations).To(HaveKeyWithValue("test", "val"))
		})

		// TODO: remove with component-cli
		It("should resolve a blueprint from a blobresolver with a gzipped blueprint", func() {
			memFs := memoryfs.New()
			err := bputils.NewBuilder().Blueprint(&lsv1alpha1.Blueprint{
				Annotations: map[string]string{
					"test": "val",
				},
			}).BuildResourceToFs(memFs, "blobs/bp.tar", true)
			Expect(err).ToNot(HaveOccurred())
			blobResolver := componentresolvers.NewLocalFilesystemBlobResolver(memFs)

			localFSAccess, err := componentresolvers.NewLocalFilesystemBlobAccess("bp.tar",
				mediatype.NewBuilder(mediatype.BlueprintArtifactsLayerMediaTypeV1).Compression(mediatype.GZipCompression).String())

			Expect(err).ToNot(HaveOccurred())

			repositoryContext := testutils.ExampleRepositoryContext()
			repositoryContexts := []*types.UnstructuredTypedObject{repositoryContext}

			cd := types.ComponentDescriptor{}
			cd.Metadata.Version = "v2"
			cd.Name = "example.com/a"
			cd.Version = "0.0.1"
			cd.Provider = "landscaper"
			cd.RepositoryContexts = repositoryContexts
			cd.Sources = []types.Source{}
			cd.Resources = append(cd.Resources, types.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-bp",
					Version: "1.2.3",
					Type:    mediatype.BlueprintType,
				},
				Relation: cdv2.ExternalRelation,
				Access:   &localFSAccess,
			})
			cd.ComponentReferences = []types.ComponentReference{}

			registryAccess, err := registries.GetFactory().NewRegistryAccess(ctx, memFs, nil, nil, nil,
				&config.LocalRegistryConfiguration{RootPath: "./blobs"}, nil, &cd, blobResolver)
			Expect(err).ToNot(HaveOccurred())

			componentVersion, err := registryAccess.GetComponentVersion(ctx, &lsv1alpha1.ComponentDescriptorReference{
				RepositoryContext: repositoryContext,
				ComponentName:     cd.GetName(),
				Version:           cd.GetVersion(),
			})
			Expect(err).NotTo(HaveOccurred())

			resource, err := componentVersion.GetResource("my-bp", nil)
			Expect(err).NotTo(HaveOccurred())
			content, err := resource.GetTypedContent(ctx)
			Expect(err).ToNot(HaveOccurred())
			bp, ok := content.Resource.(*blueprints.Blueprint)
			Expect(ok).To(BeTrue())
			Expect(bp.Info.Annotations).To(HaveKeyWithValue("test", "val"))
		})

		// TODO: remove with component-cli
		It("should throw an error if a blueprint is received corrupted", func() {
			mediaType := mediatype.NewBuilder(mediatype.BlueprintArtifactsLayerMediaTypeV1).String()

			memFs := memoryfs.New()
			err := memFs.MkdirAll("blobs", 0o777)
			Expect(err).ToNot(HaveOccurred())
			file, err := memFs.Create("blobs/bp.tar")
			Expect(err).ToNot(HaveOccurred())
			_, err = file.Write(append([]byte(`this is not a valid blueprint`), make([]byte, 1024)...))
			Expect(err).ToNot(HaveOccurred())
			blobResolver := componentresolvers.NewLocalFilesystemBlobResolver(memFs)

			localFSAccess, err := componentresolvers.NewLocalFilesystemBlobAccess("bp.tar", mediaType)
			Expect(err).ToNot(HaveOccurred())

			repositoryContext := testutils.ExampleRepositoryContext()
			repositoryContexts := []*types.UnstructuredTypedObject{repositoryContext}

			cd := types.ComponentDescriptor{}
			cd.Metadata.Version = "v2"
			cd.Name = "example.com/a"
			cd.Version = "0.0.1"
			cd.Provider = "landscaper"
			cd.RepositoryContexts = repositoryContexts
			cd.Sources = []types.Source{}
			cd.Resources = append(cd.Resources, types.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-bp",
					Version: "1.2.3",
					Type:    mediatype.BlueprintType,
				},
				Relation: cdv2.ExternalRelation,
				Access:   &localFSAccess,
			})
			cd.ComponentReferences = []types.ComponentReference{}

			registryAccess, err := registries.GetFactory().NewRegistryAccess(ctx, memFs, nil, nil, nil,
				&config.LocalRegistryConfiguration{RootPath: "./blobs"}, nil, &cd, blobResolver)
			Expect(err).ToNot(HaveOccurred())

			componentVersion, err := registryAccess.GetComponentVersion(ctx, &lsv1alpha1.ComponentDescriptorReference{
				RepositoryContext: repositoryContext,
				ComponentName:     cd.GetName(),
				Version:           cd.GetVersion(),
			})
			Expect(err).NotTo(HaveOccurred())

			resource, err := componentVersion.GetResource("my-bp", nil)
			Expect(err).NotTo(HaveOccurred())

			_, err = resource.GetTypedContent(ctx)
			Expect(err).To(HaveOccurred())
		})

		// TODO: remove with component-cli
		It("should throw an error if a blueprint is received corrupted with gzipped media type", func() {
			mediaType := mediatype.NewBuilder(mediatype.BlueprintArtifactsLayerMediaTypeV1).Compression(mediatype.GZipCompression).String()

			memFs := memoryfs.New()
			err := memFs.MkdirAll("blobs", 0o777)
			Expect(err).ToNot(HaveOccurred())
			file, err := memFs.Create("blobs/bp.tar")
			Expect(err).ToNot(HaveOccurred())
			_, err = file.Write(append([]byte(`this is not a valid blueprint`), make([]byte, 1024)...))
			Expect(err).ToNot(HaveOccurred())
			blobResolver := componentresolvers.NewLocalFilesystemBlobResolver(memFs)

			localFSAccess, err := componentresolvers.NewLocalFilesystemBlobAccess("bp.tar", mediaType)
			Expect(err).ToNot(HaveOccurred())

			repositoryContext := testutils.ExampleRepositoryContext()
			repositoryContexts := []*types.UnstructuredTypedObject{repositoryContext}

			cd := types.ComponentDescriptor{}
			cd.Metadata.Version = "v2"
			cd.Name = "example.com/a"
			cd.Version = "0.0.1"
			cd.Provider = "landscaper"
			cd.RepositoryContexts = repositoryContexts
			cd.Sources = []types.Source{}
			cd.Resources = append(cd.Resources, types.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-bp",
					Version: "1.2.3",
					Type:    mediatype.BlueprintType,
				},
				Relation: cdv2.ExternalRelation,
				Access:   &localFSAccess,
			})
			cd.ComponentReferences = []types.ComponentReference{}

			registryAccess, err := registries.GetFactory().NewRegistryAccess(ctx, memFs, nil, nil, nil,
				&config.LocalRegistryConfiguration{RootPath: "./blobs"}, nil, &cd, blobResolver)
			Expect(err).ToNot(HaveOccurred())

			componentVersion, err := registryAccess.GetComponentVersion(ctx, &lsv1alpha1.ComponentDescriptorReference{
				RepositoryContext: repositoryContext,
				ComponentName:     cd.GetName(),
				Version:           cd.GetVersion(),
			})
			Expect(err).NotTo(HaveOccurred())

			resource, err := componentVersion.GetResource("my-bp", nil)
			Expect(err).NotTo(HaveOccurred())

			_, err = resource.GetTypedContent(ctx)
			Expect(err).To(HaveOccurred())
		})

	})

})
