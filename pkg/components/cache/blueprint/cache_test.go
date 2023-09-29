// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprint_test

import (
	"context"
	"os"

	"github.com/mandelsoft/vfs/pkg/osfs"

	. "github.com/gardener/landscaper/pkg/components/cache/blueprint"
	"github.com/gardener/landscaper/pkg/components/model"

	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/apis/config/v1alpha1"
	"github.com/gardener/landscaper/apis/mediatype"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils/simplelogger"
)

const (
	BLUEPRINT_ID             = "testid"
	TESTDATA_PATH            = "testdata"
	BLUEPRINT_SUBPATH        = "blueprint"
	BLUEPRINT_EDITED_SUBPATH = "blueprint_edited"
)

// TODO
var _ = XDescribe("Put", func() {

	var defaultStoreConfig config.BlueprintStore

	BeforeEach(func() {
		cs := v1alpha1.BlueprintStore{}
		v1alpha1.SetDefaults_BlueprintStore(&cs)
		Expect(v1alpha1.Convert_v1alpha1_BlueprintStore_To_config_BlueprintStore(&cs, &defaultStoreConfig, nil)).To(Succeed())
	})

	It("should be nil if blueprint is not stored", func() {
		ctx := context.Background()
		memFs := memoryfs.New()
		store, err := NewStore(logging.Discard(), memFs, defaultStoreConfig)
		Expect(err).ToNot(HaveOccurred())

		bp, err := store.Get(ctx, BLUEPRINT_ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(bp).To(BeNil())
	})

	It("should store and retrieve the stored blueprint", func() {
		ctx := context.Background()
		memFs := memoryfs.New()
		store, err := NewStore(logging.Discard(), memFs, defaultStoreConfig)
		Expect(err).ToNot(HaveOccurred())

		fs := memoryfs.New()
		err = vfs.CopyDir(osfs.New(), TESTDATA_PATH, fs, "/")
		Expect(err).ToNot(HaveOccurred())

		bp, err := BuildBlueprintFromPath(fs, BLUEPRINT_SUBPATH)
		Expect(err).ToNot(HaveOccurred())

		ok, err := store.Put(ctx, BLUEPRINT_ID, &model.TypedResourceContent{
			Type:     mediatype.BlueprintType,
			Resource: bp,
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())

		bpFromCache, err := store.Get(ctx, BLUEPRINT_ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(bpFromCache.Info.Annotations).To(HaveKeyWithValue("test", "original"))
	})

	It("should update the blueprint if same id", func() {
		ctx := context.Background()
		memFs := memoryfs.New()
		store, err := NewStore(logging.Discard(), memFs, defaultStoreConfig)
		Expect(err).ToNot(HaveOccurred())

		fs := memoryfs.New()
		err = vfs.CopyDir(osfs.New(), TESTDATA_PATH, fs, "/")
		Expect(err).ToNot(HaveOccurred())

		bp, err := BuildBlueprintFromPath(fs, BLUEPRINT_SUBPATH)
		Expect(err).ToNot(HaveOccurred())

		ok, err := store.Put(ctx, BLUEPRINT_ID, &model.TypedResourceContent{
			Type:     mediatype.BlueprintType,
			Resource: bp,
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())

		bpFromCache, err := store.Get(ctx, BLUEPRINT_ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(bpFromCache.Info.Annotations).To(HaveKeyWithValue("test", "original"))

		bp, err = BuildBlueprintFromPath(fs, BLUEPRINT_EDITED_SUBPATH)
		Expect(err).ToNot(HaveOccurred())

		ok, err = store.Put(ctx, BLUEPRINT_ID, &model.TypedResourceContent{
			Type:     mediatype.BlueprintType,
			Resource: bp,
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(ok).To(BeTrue())

		bpFromCache, err = store.Get(ctx, BLUEPRINT_ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(bpFromCache.Info.Annotations).To(HaveKeyWithValue("test", "different"))
	})

	Context("cache", func() {
		It("should not try to get the blueprint from the store if the cache is disabled", func() {
			ctx := context.Background()
			memFs := memoryfs.New()
			defaultStoreConfig.DisableCache = true
			store, err := NewStore(logging.Discard(), memFs, defaultStoreConfig)
			Expect(err).ToNot(HaveOccurred())

			fs := memoryfs.New()
			err = vfs.CopyDir(osfs.New(), TESTDATA_PATH, fs, "/")
			Expect(err).ToNot(HaveOccurred())

			bp, err := BuildBlueprintFromPath(fs, BLUEPRINT_SUBPATH)
			Expect(err).ToNot(HaveOccurred())

			ok, err := store.Put(ctx, BLUEPRINT_ID, &model.TypedResourceContent{
				Type:     mediatype.BlueprintType,
				Resource: bp,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())

			bpFromCache, err := store.Get(ctx, BLUEPRINT_ID)
			Expect(err).To(Equal(NotFoundError))
			Expect(bpFromCache).To(BeNil())
		})

	})

	Context("GarbageCollection", func() {
		It("should delete entry with the least hits", func() {
			// it will NOT always delete the one with the least hits, the deletion priority calculation is a weighted
			// function cosidering hits and how old it is (thus, the cache will not become blocked by old entries with
			// several hits)

			ctx := context.Background()
			defaultStoreConfig.Size = "1Ki"
			store, err := NewStore(logging.Wrap(simplelogger.NewIOLogger(GinkgoWriter)), memoryfs.New(), defaultStoreConfig)
			Expect(err).ToNot(HaveOccurred())

			fs := memoryfs.New()
			err = vfs.CopyDir(osfs.New(), TESTDATA_PATH, fs, "/")
			Expect(err).ToNot(HaveOccurred())

			// add a dummy data file to the blueprint to fill the cache
			Expect(vfs.WriteFile(fs, BLUEPRINT_SUBPATH+"/test/data", make([]byte, 200), os.ModePerm)).ToNot(HaveOccurred())
			bp, err := BuildBlueprintFromPath(fs, BLUEPRINT_SUBPATH)
			Expect(err).ToNot(HaveOccurred())

			ok, err := store.Put(ctx, "a", &model.TypedResourceContent{
				Type:     mediatype.BlueprintType,
				Resource: bp,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())

			// increase hits on index a
			bpA, err := store.Get(ctx, "a")
			Expect(err).To(BeNil())
			Expect(bpA.Info.Annotations).To(HaveKeyWithValue("test", "original"))

			// create a dummy data object to fill the cache
			Expect(vfs.WriteFile(fs, BLUEPRINT_EDITED_SUBPATH+"/test/data", make([]byte, 200), os.ModePerm)).ToNot(HaveOccurred())
			bp, err = BuildBlueprintFromPath(fs, BLUEPRINT_EDITED_SUBPATH)
			Expect(err).ToNot(HaveOccurred())

			ok, err = store.Put(ctx, "b", &model.TypedResourceContent{
				Type:     mediatype.BlueprintType,
				Resource: bp,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())

			// the garbage collection should happen during each store
			// But as the gc process is async lets explicitly run it again to not have to wait here
			store.RunGarbageCollection()

			// the second one is gc'ed because it has fewer hits
			bp, err = store.Get(ctx, "b")
			Expect(err).ToNot(HaveOccurred())
			Expect(bp).To(BeNil())

			bp, err = store.Get(ctx, "a")
			Expect(err).ToNot(HaveOccurred())
			Expect(bp).ToNot(BeNil())
			Expect(bp.Info.Annotations).To(HaveKeyWithValue("test", "original"))
		})

		It("should delete the oldest entry (if all have equal hits)", func() {
			ctx := context.Background()
			defaultStoreConfig.Size = "1Ki"
			store, err := NewStore(logging.Wrap(simplelogger.NewIOLogger(GinkgoWriter)), memoryfs.New(), defaultStoreConfig)
			Expect(err).ToNot(HaveOccurred())

			fs := memoryfs.New()
			err = vfs.CopyDir(osfs.New(), TESTDATA_PATH, fs, "/")
			Expect(err).ToNot(HaveOccurred())

			// add a dummy data file to the blueprint to fill the cache
			Expect(vfs.WriteFile(fs, BLUEPRINT_SUBPATH+"/test/data", make([]byte, 200), os.ModePerm)).ToNot(HaveOccurred())
			bp, err := BuildBlueprintFromPath(fs, BLUEPRINT_SUBPATH)
			Expect(err).ToNot(HaveOccurred())

			ok, err := store.Put(ctx, "a", &model.TypedResourceContent{
				Type:     mediatype.BlueprintType,
				Resource: bp,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())

			// create a dummy data object to fill the cache
			Expect(vfs.WriteFile(fs, BLUEPRINT_EDITED_SUBPATH+"/test/data", make([]byte, 200), os.ModePerm)).ToNot(HaveOccurred())
			bp, err = BuildBlueprintFromPath(fs, BLUEPRINT_EDITED_SUBPATH)
			Expect(err).ToNot(HaveOccurred())

			ok, err = store.Put(ctx, "b", &model.TypedResourceContent{
				Type:     mediatype.BlueprintType,
				Resource: bp,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())

			// the garbage collection should happen during each store
			// But as the gc process is async lets explicitly run it again to not have to wait here
			store.RunGarbageCollection()

			// the first one is gc'ed because it is the oldest entry and the hits of blueprint 1 and blueprint 2 equal
			bp, err = store.Get(ctx, "b")
			Expect(err).ToNot(HaveOccurred())
			Expect(bp).ToNot(BeNil())
			Expect(bp.Info.Annotations).To(HaveKeyWithValue("test", "different"))

			bp, err = store.Get(ctx, "a")
			Expect(bp).To(BeNil())
			Expect(err).To(BeNil())
		})
	})

})
