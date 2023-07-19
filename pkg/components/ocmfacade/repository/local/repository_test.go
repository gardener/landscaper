package local_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gardener/landscaper/pkg/components/ocmfacade/repository"
	"github.com/gardener/landscaper/pkg/components/ocmfacade/repository/attrs/localrootfs"
	"github.com/gardener/landscaper/pkg/components/ocmfacade/repository/local"
	"github.com/mandelsoft/filepath/pkg/filepath"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/open-component-model/ocm/pkg/env/builder"
	. "github.com/open-component-model/ocm/pkg/testutils"

	"github.com/open-component-model/ocm/pkg/common/compression"
	tenv "github.com/open-component-model/ocm/pkg/env"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils/tarutils"
)

const (
	// Random Test Names
	DESCRIPTOR_PATH = "testdescriptorpath"
	BLOB_PATH       = "testblobpath"

	// Mock Directory Paths
	TAR_REPOSITORY       = "testdata/tar"
	DISTINCT_REPOSITORY  = "testdata/distinct"
	DIRECTORY_REPOSITORY = "testdata/directory"

	// Component Names
	COMPONENT_NAME    = "example.com/root"
	COMPONENT_VERSION = "1.0.0"
	RESOURCE_NAME     = "test"
)

var _ = Describe("ocm-lib based landscaper local repository", func() {
	var env *Builder
	var fs vfs.FileSystem

	BeforeEach(func() {
		env = NewBuilder(tenv.NewEnvironment())
		fs = Must(projectionfs.New(osfs.New(), ""))
	})

	AfterEach(func() {
		env.Cleanup()
	})

	It("marshal/unmarshal spec v1", func() {
		spec := Must(local.NewRepositorySpecV1(DESCRIPTOR_PATH))
		Expect(spec).ToNot(BeNil())
		Expect(spec.CompDescFs).To(BeNil())
		Expect(spec.BlobFs).To(BeNil())

		spec = Must(local.NewRepositorySpecV1(DESCRIPTOR_PATH, fs))
		Expect(spec).ToNot(BeNil())
		Expect(spec.CompDescFs).To(Equal(fs))
		Expect(spec.BlobFs).To(Equal(fs))

		data := Must(json.Marshal(spec))
		Expect(string(data)).To(Equal(fmt.Sprintf("{\"type\":\"%s\",\"filePath\":\"%s\"}", local.Type, DESCRIPTOR_PATH)))
		spec1 := Must(env.OCMContext().RepositorySpecForConfig(data, runtime.DefaultJSONEncoding)).(*repository.RepositorySpec)
		// spec will not completely equal spec1 as the filesystem cannot be serialized
		Expect(spec1.Type).To(Equal(spec.Type))
		Expect(spec1.CompDescDirPath).To(Equal(spec.CompDescDirPath))
		Expect(spec1.BlobDirPath).To(Equal(spec.BlobDirPath))
	})

	It("marshal/unmarshal spec v2", func() {
		spec := Must(local.NewRepositorySpecV2(nil, DESCRIPTOR_PATH, BLOB_PATH))
		Expect(spec).ToNot(BeNil())
		Expect(spec.CompDescFs).To(BeNil())
		Expect(spec.BlobFs).To(BeNil())

		spec = Must(local.NewRepositorySpecV2(fs, DESCRIPTOR_PATH, BLOB_PATH))
		Expect(spec).ToNot(BeNil())
		Expect(spec.CompDescFs).To(Equal(fs))
		Expect(spec.BlobFs).To(Equal(fs))

		data := Must(json.Marshal(spec))
		Expect(string(data)).To(Equal(fmt.Sprintf("{\"type\":\"%s\",\"compDescDirPath\":\"%s\",\"blobDirPath\":\"%s\"}", local.TypeV2, DESCRIPTOR_PATH, BLOB_PATH)))
		spec1 := Must(env.OCMContext().RepositorySpecForConfig(data, runtime.DefaultJSONEncoding)).(*repository.RepositorySpec)
		// spec will not completely equal spec1 as the filesystem cannot be serialized
		Expect(spec1.Type).To(Equal(spec.Type))
		Expect(spec1.CompDescDirPath).To(Equal(spec.CompDescDirPath))
		Expect(spec1.BlobDirPath).To(Equal(spec.BlobDirPath))
	})

	It("repository (from spec v1) with resource stored as tar in blob directory", func() {
		// this use case pretty much resembles a component archive

		// this has to be set if the PathFileSystem within the spec is not set
		// as a filesystem cannot really be serialized, this is always the case if the spec is created from a serialized
		// form (e.g. coming from the installation)
		localrootfs.Set(env.OCMContext(), fs)
		spec := Must(local.NewRepositorySpecV1(TAR_REPOSITORY))
		repo := Must(spec.Repository(env.OCMContext(), nil))
		defer Close(repo)
		cv := Must(repo.LookupComponentVersion(COMPONENT_NAME, COMPONENT_VERSION))
		defer Close(cv)
		res := Must(cv.GetResourcesByName(RESOURCE_NAME))
		acc := Must(res[0].AccessMethod())
		defer Close(acc)
		data := Must(acc.Reader())
		defer Close(data)

		mfs := memoryfs.New()
		data, _ = Must2(compression.AutoDecompress(data))
		_, _ = Must2(tarutils.ExtractTarToFsWithInfo(mfs, data))
		bytesA := []byte{}
		_ = Must(Must(mfs.Open("testfile")).Read(bytesA))

		bytesB := []byte{}
		_ = Must(Must(osfs.New().Open(TAR_REPOSITORY + "/blobs/sha256.3ed99e50092c619823e2c07941c175ea2452f1455f570c55510586b387ec2ff2")).Read(bytesB))
		bufferB := bytes.NewBuffer(bytesB)
		r, _ := Must2(compression.AutoDecompress(bufferB))
		_, _ = Must2(tarutils.ExtractTarToFsWithInfo(mfs, r))
		Expect(bytesA).To(Equal(bufferB.Bytes()))
	})

	It("repository (from spec v2) with component descriptors and resources stored in distinct directories", func() {
		localrootfs.Set(env.OCMContext(), fs)
		spec := Must(local.NewRepositorySpecV2(fs, filepath.Join(DISTINCT_REPOSITORY, "compdescs"), filepath.Join(DISTINCT_REPOSITORY, "blobs")))
		repo := Must(spec.Repository(env.OCMContext(), nil))
		defer Close(repo)
		cv := Must(repo.LookupComponentVersion(COMPONENT_NAME, COMPONENT_VERSION))
		defer Close(cv)
		res := Must(cv.GetResourcesByName(RESOURCE_NAME))
		acc := Must(res[0].AccessMethod())
		defer Close(acc)
		bufferA := Must(acc.Get())

		bufferB := Must(vfs.ReadFile(fs, filepath.Join(DISTINCT_REPOSITORY, "blobs", "blob1")))
		Expect(bufferA).To(Equal(bufferB))
	})

	It("repository (from spec v2) with a directory resource", func() {
		localrootfs.Set(env.OCMContext(), fs)
		spec := Must(local.NewRepositorySpecV2(fs, DIRECTORY_REPOSITORY, DIRECTORY_REPOSITORY))
		repo := Must(spec.Repository(env.OCMContext(), nil))
		defer Close(repo)
		cv := Must(repo.LookupComponentVersion(COMPONENT_NAME, COMPONENT_VERSION))
		defer Close(cv)
		res := Must(cv.GetResourcesByName(RESOURCE_NAME))
		acc := Must(res[0].AccessMethod())
		defer Close(acc)
		data := Must(acc.Reader())
		defer Close(data)

		mfs := memoryfs.New()
		_, _, err := tarutils.ExtractTarToFsWithInfo(mfs, data)
		Expect(err).ToNot(HaveOccurred())
		bufferA := Must(vfs.ReadFile(mfs, "testblob"))
		bufferB := Must(vfs.ReadFile(fs, filepath.Join(DIRECTORY_REPOSITORY, "blob1", "testblob")))
		Expect(bufferA).To(Equal(bufferB))
	})
})
