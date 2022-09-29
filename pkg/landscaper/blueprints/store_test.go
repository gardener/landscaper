// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints_test

import (
	"context"
	"io"
	"os"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/apis/mediatype"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"

	"github.com/gardener/landscaper/pkg/utils/simplelogger"

	"github.com/gardener/landscaper/apis/config/v1alpha1"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints/bputils"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
)

var _ = Describe("Store", func() {

	var defaultStoreConfig config.BlueprintStore

	BeforeEach(func() {
		cs := v1alpha1.BlueprintStore{}
		v1alpha1.SetDefaults_BlueprintStore(&cs)
		Expect(v1alpha1.Convert_v1alpha1_BlueprintStore_To_config_BlueprintStore(&cs, &defaultStoreConfig, nil)).To(Succeed())
	})

	It("should throw an error if a blueprint is not stored", func() {
		ctx := context.Background()
		memFs := memoryfs.New()
		store, err := blueprints.NewStore(logging.Discard(), memFs, defaultStoreConfig)
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		cd.Name = "example.com/a"
		cd.Version = "0.0.1"
		res := cdv2.Resource{}
		res.Name = "blueprint"
		res.Version = "0.0.2"
		_, err = store.Get(ctx, "a")
		Expect(err).To(Equal(blueprints.NotFoundError))
	})

	It("should store and retrieve the stored blueprint", func() {
		ctx := context.Background()
		memFs := memoryfs.New()
		store, err := blueprints.NewStore(logging.Discard(), memFs, defaultStoreConfig)
		Expect(err).ToNot(HaveOccurred())

		data, blobInfo, err := bputils.NewBuilder().Blueprint(&lsv1alpha1.Blueprint{
			Annotations: map[string]string{
				"test": "val",
			},
		}).BuildResource(false)
		Expect(err).ToNot(HaveOccurred())
		defer data.Close()

		cd := &cdv2.ComponentDescriptor{}
		cd.Name = "example.com/a"
		cd.Version = "0.0.1"
		res := cdv2.Resource{}
		res.Name = "blueprint"
		res.Version = "0.0.2"
		_, err = store.Store(ctx, defaultBlobResolver(data, blobInfo), res, "a", nil)
		Expect(err).ToNot(HaveOccurred())

		bp, err := store.Get(ctx, "a")
		Expect(err).ToNot(HaveOccurred())
		Expect(bp.Info.Annotations).To(HaveKeyWithValue("test", "val"))
	})

	Context("Fetch", func() {

		It("should store and retrieve the stored blueprint", func() {
			ctx := context.Background()
			memFs := memoryfs.New()
			store, err := blueprints.NewStore(logging.Discard(), memFs, defaultStoreConfig)
			Expect(err).ToNot(HaveOccurred())

			data, blobInfo, err := bputils.NewBuilder().Blueprint(&lsv1alpha1.Blueprint{
				Annotations: map[string]string{
					"test": "val",
				},
			}).BuildResource(false)
			Expect(err).ToNot(HaveOccurred())
			defer data.Close()

			cd := &cdv2.ComponentDescriptor{}
			cd.Name = "example.com/a"
			cd.Version = "0.0.1"
			res := cdv2.Resource{}
			res.Name = "blueprint"
			res.Version = "0.0.2"
			res.Type = mediatype.BlueprintType
			cd.Resources = append(cd.Resources, res)
			_, err = store.Fetch(ctx, cd, defaultBlobResolver(data, blobInfo), "blueprint")
			Expect(err).ToNot(HaveOccurred())

			bp, err := store.Fetch(ctx, cd, defaultBlobResolver(data, blobInfo), "blueprint")
			Expect(err).ToNot(HaveOccurred())
			Expect(bp.Info.Annotations).To(HaveKeyWithValue("test", "val"))
		})

		It("should not update the stored blueprint if component descriptor and blueprint are immutable (indexmethod component descriptor)", func() {
			ctx := context.Background()
			memFs := memoryfs.New()
			defaultStoreConfig.IndexMethod = config.ComponentDescriptorIdentityMethod
			store, err := blueprints.NewStore(logging.Discard(), memFs, defaultStoreConfig)
			Expect(err).ToNot(HaveOccurred())

			data, blobInfo, err := bputils.NewBuilder().Blueprint(&lsv1alpha1.Blueprint{
				Annotations: map[string]string{
					"test": "val",
				},
			}).BuildResource(false)
			Expect(err).ToNot(HaveOccurred())
			defer data.Close()

			cd := &cdv2.ComponentDescriptor{}
			cd.Name = "example.com/a"
			cd.Version = "0.0.1"
			res := cdv2.Resource{}
			res.Name = "blueprint"
			res.Version = "0.0.2"
			res.Type = mediatype.BlueprintType
			cd.Resources = append(cd.Resources, res)
			_, err = store.Fetch(ctx, cd, defaultBlobResolver(data, blobInfo), "blueprint")
			Expect(err).ToNot(HaveOccurred())

			data2, blobInfo2, err := bputils.NewBuilder().Blueprint(&lsv1alpha1.Blueprint{
				Annotations: map[string]string{
					"test": "val2",
				},
			}).BuildResource(false)
			Expect(err).ToNot(HaveOccurred())
			defer data.Close()

			bp, err := store.Fetch(ctx, cd, defaultBlobResolver(data2, blobInfo2), "blueprint")
			Expect(err).ToNot(HaveOccurred())
			Expect(bp.Info.Annotations).To(HaveKeyWithValue("test", "val"))
		})

		It("should update the stored blueprint if blueprint is indexed by its digest", func() {
			ctx := context.Background()
			memFs := memoryfs.New()
			defaultStoreConfig.IndexMethod = config.BlueprintDigestIndex
			store, err := blueprints.NewStore(logging.Discard(), memFs, defaultStoreConfig)
			Expect(err).ToNot(HaveOccurred())

			data, blobInfo, err := bputils.NewBuilder().Blueprint(&lsv1alpha1.Blueprint{
				Annotations: map[string]string{
					"test": "val",
				},
			}).BuildResource(false)
			Expect(err).ToNot(HaveOccurred())
			defer data.Close()

			cd := &cdv2.ComponentDescriptor{}
			cd.Name = "example.com/a"
			cd.Version = "0.0.1"
			res := cdv2.Resource{}
			res.Name = "blueprint"
			res.Version = "0.0.2"
			res.Type = mediatype.BlueprintType
			cd.Resources = append(cd.Resources, res)
			_, err = store.Fetch(ctx, cd, defaultBlobResolver(data, blobInfo), "blueprint")
			Expect(err).ToNot(HaveOccurred())

			data2, blobInfo2, err := bputils.NewBuilder().Blueprint(&lsv1alpha1.Blueprint{
				Annotations: map[string]string{
					"test": "val2",
				},
			}).BuildResource(false)
			Expect(err).ToNot(HaveOccurred())
			defer data.Close()

			bp, err := store.Fetch(ctx, cd, defaultBlobResolver(data2, blobInfo2), "blueprint")
			Expect(err).ToNot(HaveOccurred())
			Expect(bp.Info.Annotations).To(HaveKeyWithValue("test", "val2"))
		})

	})

	Context("cache", func() {
		It("should not try to get the blueprint from the store if the cache is disabled", func() {
			ctx := context.Background()
			memFs := memoryfs.New()
			defaultStoreConfig.DisableCache = true
			store, err := blueprints.NewStore(logging.Discard(), memFs, defaultStoreConfig)
			Expect(err).ToNot(HaveOccurred())

			data, blobInfo, err := bputils.NewBuilder().Blueprint(&lsv1alpha1.Blueprint{
				Annotations: map[string]string{
					"test": "val",
				},
			}).BuildResource(false)
			Expect(err).ToNot(HaveOccurred())
			defer data.Close()

			cd := &cdv2.ComponentDescriptor{}
			cd.Name = "example.com/a"
			cd.Version = "0.0.1"
			res := cdv2.Resource{}
			res.Name = "blueprint"
			res.Version = "0.0.2"
			_, err = store.Store(ctx, defaultBlobResolver(data, blobInfo), res, "a", nil)
			Expect(err).ToNot(HaveOccurred())

			_, err = store.Get(ctx, "a")
			Expect(err).To(Equal(blueprints.NotFoundError))
		})

		It("should not update a blueprint in the store if the cache is enabled", func() {
			ctx := context.Background()
			memFs := memoryfs.New()
			defaultStoreConfig.DisableCache = false
			store, err := blueprints.NewStore(logging.Discard(), memFs, defaultStoreConfig)
			Expect(err).ToNot(HaveOccurred())

			data, blobInfo, err := bputils.NewBuilder().Blueprint(&lsv1alpha1.Blueprint{
				Annotations: map[string]string{
					"test": "val",
				},
			}).BuildResource(false)
			Expect(err).ToNot(HaveOccurred())
			defer data.Close()

			cd := &cdv2.ComponentDescriptor{}
			cd.Name = "example.com/a"
			cd.Version = "0.0.1"
			res := cdv2.Resource{}
			res.Name = "blueprint"
			res.Version = "0.0.2"
			bp, err := store.Store(ctx, defaultBlobResolver(data, blobInfo), res, "a", nil)
			Expect(err).ToNot(HaveOccurred())

			Expect(bp.Info.Annotations).To(HaveKeyWithValue("test", "val"))

			data, blobInfo, err = bputils.NewBuilder().Blueprint(&lsv1alpha1.Blueprint{
				Annotations: map[string]string{
					"test": "val2",
				},
			}).BuildResource(false)
			Expect(err).ToNot(HaveOccurred())
			defer data.Close()

			bp, err = store.Store(ctx, defaultBlobResolver(data, blobInfo), res, "a", nil)
			Expect(err).ToNot(HaveOccurred())

			Expect(bp.Info.Annotations).To(HaveKeyWithValue("test", "val"))
		})

		It("should update a blueprint in the store if the cache is disabled", func() {
			ctx := context.Background()
			memFs := memoryfs.New()
			defaultStoreConfig.DisableCache = true
			store, err := blueprints.NewStore(logging.Discard(), memFs, defaultStoreConfig)
			Expect(err).ToNot(HaveOccurred())

			data, blobInfo, err := bputils.NewBuilder().Blueprint(&lsv1alpha1.Blueprint{
				Annotations: map[string]string{
					"test": "val",
				},
			}).BuildResource(false)
			Expect(err).ToNot(HaveOccurred())
			defer data.Close()

			cd := &cdv2.ComponentDescriptor{}
			cd.Name = "example.com/a"
			cd.Version = "0.0.1"
			res := cdv2.Resource{}
			res.Name = "blueprint"
			res.Version = "0.0.2"
			bp, err := store.Store(ctx, defaultBlobResolver(data, blobInfo), res, "a", nil)
			Expect(err).ToNot(HaveOccurred())

			Expect(bp.Info.Annotations).To(HaveKeyWithValue("test", "val"))

			data, blobInfo, err = bputils.NewBuilder().Blueprint(&lsv1alpha1.Blueprint{
				Annotations: map[string]string{
					"test": "val2",
				},
			}).BuildResource(false)
			Expect(err).ToNot(HaveOccurred())
			defer data.Close()

			bp, err = store.Store(ctx, defaultBlobResolver(data, blobInfo), res, "a", nil)
			Expect(err).ToNot(HaveOccurred())

			Expect(bp.Info.Annotations).To(HaveKeyWithValue("test", "val2"))
		})
	})

	Context("GarbageCollection", func() {
		It("should store and retrieve the stored blueprint", func() {
			ctx := context.Background()
			defaultStoreConfig.Size = "1Ki"
			store, err := blueprints.NewStore(logging.Wrap(simplelogger.NewIOLogger(GinkgoWriter)), memoryfs.New(), defaultStoreConfig)
			Expect(err).ToNot(HaveOccurred())

			bpFs := memoryfs.New()
			// create a dummy data object to fill the cache
			Expect(vfs.WriteFile(bpFs, "data", make([]byte, 700), os.ModePerm))
			data, blobInfo, err := bputils.NewBuilder().Fs(bpFs).Blueprint(&lsv1alpha1.Blueprint{
				Annotations: map[string]string{
					"test": "val",
				},
			}).BuildResource(false)
			Expect(err).ToNot(HaveOccurred())

			cd := &cdv2.ComponentDescriptor{}
			cd.Name = "example.com/a"
			cd.Version = "0.0.1"
			res := cdv2.Resource{}
			res.Name = "blueprint"
			res.Version = "0.0.2"
			_, err = store.Store(ctx, defaultBlobResolver(data, blobInfo), res, "a", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(data.Close()).To(Succeed())

			bpFs2 := memoryfs.New()
			// create a dummy data object to fill the cache
			Expect(vfs.WriteFile(bpFs2, "data", make([]byte, 700), os.ModePerm))
			data2, blobInfo2, err := bputils.NewBuilder().Fs(bpFs2).Blueprint(&lsv1alpha1.Blueprint{
				Annotations: map[string]string{
					"test2": "val",
				},
			}).BuildResource(false)
			Expect(err).ToNot(HaveOccurred())
			res2 := cdv2.Resource{}
			res2.Name = "blueprint2"
			res2.Version = "0.0.2"
			_, err = store.Store(ctx, defaultBlobResolver(data2, blobInfo2), res, "b", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(data2.Close()).To(Succeed())

			// the garbage collection should happen during each store
			// But as the gc process is async lets explicitly run it again to not have to wait here
			store.RunGarbageCollection()

			// the first one is gc'ed because it is the oldest entry and the hits of blueprint 1 and blueprint 2 equal
			bp, err := store.Get(ctx, "b")
			Expect(err).ToNot(HaveOccurred())
			Expect(bp.Info.Annotations).To(HaveKeyWithValue("test2", "val"))

			_, err = store.Get(ctx, "a")
			Expect(err).To(Equal(blueprints.NotFoundError))
		})
	})

})

func defaultBlobResolver(data io.Reader, blobInfo *ctf.BlobInfo) ctf.BlobResolver {
	return cdutils.NewBlobResolverFunc(func(ctx context.Context, res cdv2.Resource, writer io.Writer) (*ctf.BlobInfo, error) {
		if _, err := io.Copy(writer, data); err != nil {
			return nil, err
		}
		return blobInfo, nil
	}).WithInfo(func(ctx context.Context, res cdv2.Resource) (*ctf.BlobInfo, error) {
		return blobInfo, nil
	})
}
