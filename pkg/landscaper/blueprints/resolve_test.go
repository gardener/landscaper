// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints_test

import (
	"context"
	"crypto/rand"
	"github.com/gardener/landscaper/pkg/components/registries"
	"io"

	"github.com/gardener/landscaper/pkg/components/cache/blueprint"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/apis/config/v1alpha1"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/mediatype"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/components/cnudie/componentresolvers"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints/bputils"
	testutils "github.com/gardener/landscaper/test/utils"
)

type dummyBlobResolver struct {
	mediaType string
}

func newDummyBlobResolver(mediaType string) ctf.BlobResolver {
	return dummyBlobResolver{
		mediaType: mediaType,
	}
}

func (r dummyBlobResolver) Info(_ context.Context, _ types.Resource) (*ctf.BlobInfo, error) {
	return &ctf.BlobInfo{
		MediaType: r.mediaType,
	}, nil
}

func (r dummyBlobResolver) Resolve(_ context.Context, _ types.Resource, writer io.Writer) (*ctf.BlobInfo, error) {
	data := make([]byte, 256)
	if _, err := rand.Read(data); err != nil {
		return nil, err
	}

	for i := 0; i < 20; i++ {
		if _, err := writer.Write(data); err != nil {
			return nil, err
		}
	}
	return &ctf.BlobInfo{
		MediaType: r.mediaType,
	}, nil
}

func (r dummyBlobResolver) CanResolve(resource types.Resource) bool {
	return true
}

var _ = Describe("Resolve", func() {

	var defaultStoreConfig config.BlueprintStore

	BeforeEach(func() {
		cs := v1alpha1.BlueprintStore{}
		v1alpha1.SetDefaults_BlueprintStore(&cs)
		Expect(v1alpha1.Convert_v1alpha1_BlueprintStore_To_config_BlueprintStore(&cs, &defaultStoreConfig, nil)).To(Succeed())
	})

	Context("ResolveBlueprintFromBlobResolver", func() {

		It("should resolve a blueprint from a blobresolver", func() {
			ctx := context.Background()

			store, err := blueprint.NewStore(logging.Discard(), memoryfs.New(), defaultStoreConfig)
			Expect(err).ToNot(HaveOccurred())
			blueprint.SetStore(store)

			memFs := memoryfs.New()
			err = bputils.NewBuilder().Blueprint(&lsv1alpha1.Blueprint{
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

			registryAccess, err := registries.GetFactory().NewRegistryAccess(ctx, memFs, nil, nil,
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

		It("should resolve a blueprint from a blobresolver with a gzipped blueprint", func() {
			ctx := context.Background()

			store, err := blueprint.NewStore(logging.Discard(), memoryfs.New(), defaultStoreConfig)
			Expect(err).ToNot(HaveOccurred())
			blueprint.SetStore(store)

			memFs := memoryfs.New()
			err = bputils.NewBuilder().Blueprint(&lsv1alpha1.Blueprint{
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

			registryAccess, err := registries.GetFactory().NewRegistryAccess(ctx, memFs, nil, nil,
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

		// I do not think it is particularly useful to test for this
		//FIt("should throw an error if a gzipped blueprint is expected but a tar is given", func() {
		//	ctx := context.Background()
		//
		//	store, err := blueprint.NewStore(logging.Discard(), memoryfs.New(), defaultStoreConfig)
		//	Expect(err).ToNot(HaveOccurred())
		//	blueprint.SetStore(store)
		//
		//	memFs := memoryfs.New()
		//	err = bputils.NewBuilder().Blueprint(&lsv1alpha1.Blueprint{
		//		Annotations: map[string]string{
		//			"test": "val",
		//		},
		//	}).BuildResourceToFs(memFs, "blobs/bp.tar", false)
		//	Expect(err).ToNot(HaveOccurred())
		//	blobResolver := componentresolvers.NewLocalFilesystemBlobResolver(memFs)
		//
		//	localFSAccess, err := componentresolvers.NewLocalFilesystemBlobAccess("bp.tar",
		//		mediatype.NewBuilder(mediatype.BlueprintArtifactsLayerMediaTypeV1).Compression(mediatype.GZipCompression).String())
		//
		//	Expect(err).ToNot(HaveOccurred())
		//
		//	repositoryContext := testutils.ExampleRepositoryContext()
		//	repositoryContexts := []*types.UnstructuredTypedObject{repositoryContext}
		//
		//	cd := types.ComponentDescriptor{}
		//	cd.Metadata.Version = "v2"
		//	cd.Name = "example.com/a"
		//	cd.Version = "0.0.1"
		//	cd.Provider = "landscaper"
		//	cd.RepositoryContexts = repositoryContexts
		//	cd.Sources = []types.Source{}
		//	cd.Resources = append(cd.Resources, types.Resource{
		//		IdentityObjectMeta: cdv2.IdentityObjectMeta{
		//			Name:    "my-bp",
		//			Version: "1.2.3",
		//			Type:    mediatype.BlueprintType,
		//		},
		//		Relation: cdv2.ExternalRelation,
		//		Access:   &localFSAccess,
		//	})
		//	cd.ComponentReferences = []types.ComponentReference{}
		//
		//	registryAccess, err := registries.GetFactory().NewRegistryAccess(ctx, memFs, nil, nil,
		//		&config.LocalRegistryConfiguration{RootPath: "./blobs"}, nil, &cd, blobResolver)
		//	Expect(err).ToNot(HaveOccurred())
		//
		//	componentVersion, err := registryAccess.GetComponentVersion(ctx, &lsv1alpha1.ComponentDescriptorReference{
		//		RepositoryContext: repositoryContext,
		//		ComponentName:     cd.GetName(),
		//		Version:           cd.GetVersion(),
		//	})
		//	Expect(err).NotTo(HaveOccurred())
		//
		//	resource, err := componentVersion.GetResource("my-bp", nil)
		//	Expect(err).NotTo(HaveOccurred())
		//
		//	_, err = resource.GetTypedContent(ctx)
		//	Expect(err).To(HaveOccurred())
		//})

		It("should throw an error if a blueprint is received corrupted", func() {
			ctx := context.Background()

			store, err := blueprint.NewStore(logging.Discard(), memoryfs.New(), defaultStoreConfig)
			Expect(err).ToNot(HaveOccurred())
			blueprint.SetStore(store)

			mediaType := mediatype.NewBuilder(mediatype.BlueprintArtifactsLayerMediaTypeV1).String()

			memFs := memoryfs.New()
			err = memFs.MkdirAll("blobs", 0o777)
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

			registryAccess, err := registries.GetFactory().NewRegistryAccess(ctx, memFs, nil, nil,
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

		It("should throw an error if a blueprint is received corrupted with gzipped media type", func() {
			ctx := context.Background()

			store, err := blueprint.NewStore(logging.Discard(), memoryfs.New(), defaultStoreConfig)
			Expect(err).ToNot(HaveOccurred())
			blueprint.SetStore(store)

			mediaType := mediatype.NewBuilder(mediatype.BlueprintArtifactsLayerMediaTypeV1).Compression(mediatype.GZipCompression).String()

			memFs := memoryfs.New()
			err = memFs.MkdirAll("blobs", 0o777)
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

			registryAccess, err := registries.GetFactory().NewRegistryAccess(ctx, memFs, nil, nil,
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
