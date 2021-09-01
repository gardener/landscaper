// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints_test

import (
	"context"
	"io"
	"math/rand"
	"time"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/apis/mediatype"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"

	"github.com/gardener/landscaper/apis/config/v1alpha1"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints/bputils"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
)

type dummyBlobResolver struct {
}

func newDummyBlobResolver() ctf.BlobResolver {
	return dummyBlobResolver{}
}

func (r dummyBlobResolver) Info(_ context.Context, _ cdv2.Resource) (*ctf.BlobInfo, error) {
	return &ctf.BlobInfo{
		MediaType: mediatype.NewBuilder(mediatype.BlueprintArtifactsLayerMediaTypeV1).String(),
	}, nil
}

func (r dummyBlobResolver) Resolve(_ context.Context, _ cdv2.Resource, writer io.Writer) (*ctf.BlobInfo, error) {
	data := make([]byte, 256)
	rand.Read(data)

	for i := 0; i < 20; i++ {
		if _, err := writer.Write(data); err != nil {
			return nil, err
		}

		time.Sleep(100 * time.Millisecond)
	}
	return nil, nil
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

			store, err := blueprints.NewStore(logr.Discard(), memoryfs.New(), defaultStoreConfig)
			Expect(err).ToNot(HaveOccurred())
			blueprints.SetStore(store)

			memFs := memoryfs.New()
			err = bputils.NewBuilder().Blueprint(&lsv1alpha1.Blueprint{
				Annotations: map[string]string{
					"test": "val",
				},
			}).BuildResourceToFs(memFs, "blobs/bp.tar", false)
			Expect(err).ToNot(HaveOccurred())
			blobResolver := componentsregistry.NewLocalFilesystemBlobResolver(memFs)
			localFSAccess, err := cdv2.NewUnstructured(cdv2.NewLocalFilesystemBlobAccess("bp.tar", mediatype.BlueprintArtifactsLayerMediaTypeV1))
			Expect(err).ToNot(HaveOccurred())

			cd := &cdv2.ComponentDescriptor{}
			cd.Name = "example.com/a"
			cd.Version = "0.0.1"
			cd.Resources = append(cd.Resources, cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-bp",
					Version: "1.2.3",
					Type:    mediatype.BlueprintType,
				},
				Relation: cdv2.ExternalRelation,
				Access:   &localFSAccess,
			})

			bp, err := blueprints.ResolveBlueprintFromBlobResolver(ctx, cd, blobResolver, "my-bp")
			Expect(err).ToNot(HaveOccurred())
			Expect(bp.Info.Annotations).To(HaveKeyWithValue("test", "val"))
		})

		It("should resolve a blueprint from a blobresolver with a gzipped blueprint", func() {
			ctx := context.Background()

			store, err := blueprints.NewStore(logr.Discard(), memoryfs.New(), defaultStoreConfig)
			Expect(err).ToNot(HaveOccurred())
			blueprints.SetStore(store)

			memFs := memoryfs.New()
			err = bputils.NewBuilder().Blueprint(&lsv1alpha1.Blueprint{
				Annotations: map[string]string{
					"test": "val",
				},
			}).BuildResourceToFs(memFs, "blobs/bp.tar", true)
			Expect(err).ToNot(HaveOccurred())
			blobResolver := componentsregistry.NewLocalFilesystemBlobResolver(memFs)
			localFSAccess, err := cdv2.NewUnstructured(cdv2.NewLocalFilesystemBlobAccess("bp.tar",
				mediatype.NewBuilder(mediatype.BlueprintArtifactsLayerMediaTypeV1).Compression(mediatype.GZipCompression).String()))
			Expect(err).ToNot(HaveOccurred())

			cd := &cdv2.ComponentDescriptor{}
			cd.Name = "example.com/a"
			cd.Version = "0.0.1"
			cd.Resources = append(cd.Resources, cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-bp",
					Version: "1.2.3",
					Type:    mediatype.BlueprintType,
				},
				Relation: cdv2.ExternalRelation,
				Access:   &localFSAccess,
			})

			bp, err := blueprints.ResolveBlueprintFromBlobResolver(ctx, cd, blobResolver, "my-bp")
			Expect(err).ToNot(HaveOccurred())
			Expect(bp.Info.Annotations).To(HaveKeyWithValue("test", "val"))
		})

		It("should throw an error if a gzipped blueprint is expected but a tar is given", func() {
			ctx := context.Background()

			store, err := blueprints.NewStore(logr.Discard(), memoryfs.New(), defaultStoreConfig)
			Expect(err).ToNot(HaveOccurred())
			blueprints.SetStore(store)

			memFs := memoryfs.New()
			err = bputils.NewBuilder().Blueprint(&lsv1alpha1.Blueprint{
				Annotations: map[string]string{
					"test": "val",
				},
			}).BuildResourceToFs(memFs, "blobs/bp.tar", false)
			Expect(err).ToNot(HaveOccurred())
			blobResolver := componentsregistry.NewLocalFilesystemBlobResolver(memFs)
			localFSAccess, err := cdv2.NewUnstructured(cdv2.NewLocalFilesystemBlobAccess("bp.tar",
				mediatype.NewBuilder(mediatype.BlueprintArtifactsLayerMediaTypeV1).Compression(mediatype.GZipCompression).String()))
			Expect(err).ToNot(HaveOccurred())

			cd := &cdv2.ComponentDescriptor{}
			cd.Name = "example.com/a"
			cd.Version = "0.0.1"
			cd.Resources = append(cd.Resources, cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-bp",
					Version: "1.2.3",
					Type:    mediatype.BlueprintType,
				},
				Relation: cdv2.ExternalRelation,
				Access:   &localFSAccess,
			})

			_, err = blueprints.ResolveBlueprintFromBlobResolver(ctx, cd, blobResolver, "my-bp")
			Expect(err).To(HaveOccurred())
		})

		It("should throw an error if a blueprint is received corrupted", func() {
			ctx := context.Background()

			store, err := blueprints.NewStore(logr.Discard(), memoryfs.New(), defaultStoreConfig)
			Expect(err).ToNot(HaveOccurred())
			blueprints.SetStore(store)

			blobResolver := newDummyBlobResolver()
			localFSAccess, err := cdv2.NewUnstructured(cdv2.NewLocalFilesystemBlobAccess("bp.tar",
				mediatype.NewBuilder(mediatype.BlueprintArtifactsLayerMediaTypeV1).String()))
			Expect(err).ToNot(HaveOccurred())

			cd := &cdv2.ComponentDescriptor{}
			cd.Name = "example.com/a"
			cd.Version = "0.0.1"
			cd.Resources = append(cd.Resources, cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-bp",
					Version: "1.2.3",
					Type:    mediatype.BlueprintType,
				},
				Relation: cdv2.ExternalRelation,
				Access:   &localFSAccess,
			})

			_, err = blueprints.ResolveBlueprintFromBlobResolver(ctx, cd, blobResolver, "my-bp")
			Expect(err).To(HaveOccurred())
		})

	})

})
