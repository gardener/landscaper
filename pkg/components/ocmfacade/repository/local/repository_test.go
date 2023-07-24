package local_test

import (
	"encoding/json"
	"fmt"

	"github.com/mandelsoft/filepath/pkg/filepath"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/open-component-model/ocm/pkg/env/builder"
	. "github.com/open-component-model/ocm/pkg/testutils"

	"github.com/gardener/landscaper/pkg/components/ocmfacade/repository"
	"github.com/gardener/landscaper/pkg/components/ocmfacade/repository/local"

	tenv "github.com/open-component-model/ocm/pkg/env"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	// Random Test Names
	DESCRIPTOR_PATH = "testdescriptorpath"

	// Mock Directory Paths
	TAR_REPOSITORY = "testdata/tar"

	// Component Names
	COMPONENT_NAME    = "example.com/root"
	COMPONENT_VERSION = "1.0.0"
	RESOURCE_NAME     = "test"
)

var _ = Describe("ocm-lib based landscaper local repository", func() {
	var env *Builder

	BeforeEach(func() {
		env = NewBuilder(tenv.NewEnvironment(tenv.TestData()))
	})

	AfterEach(func() {
		_ = env.Cleanup()
	})

	It("marshal/unmarshal spec v1", func() {
		spec := Must(local.NewRepositorySpecV1(DESCRIPTOR_PATH))
		Expect(spec).ToNot(BeNil())
		Expect(spec.FileSystem).To(BeNil())
		Expect(spec.BlobFs).To(BeNil())

		spec = Must(local.NewRepositorySpecV1(DESCRIPTOR_PATH, env))
		Expect(spec).ToNot(BeNil())
		Expect(spec.FileSystem).To(Equal(env))

		data := Must(json.Marshal(spec))
		Expect(string(data)).To(Equal(fmt.Sprintf("{\"type\":\"%s\",\"filePath\":\"%s\"}", local.Type, DESCRIPTOR_PATH)))
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
		//vfsattr.Set(env.OCMContext(), env)
		spec := Must(local.NewRepositorySpecV1(TAR_REPOSITORY, env))
		repo := Must(spec.Repository(env.OCMContext(), nil))
		defer Close(repo)
		cv := Must(repo.LookupComponentVersion(COMPONENT_NAME, COMPONENT_VERSION))
		defer Close(cv)
		res := Must(cv.GetResourcesByName(RESOURCE_NAME))
		acc := Must(res[0].AccessMethod())
		defer Close(acc)
		bytesA := Must(acc.Get())

		bytesB := Must(vfs.ReadFile(env, filepath.Join(TAR_REPOSITORY, "blobs", "sha256.3ed99e50092c619823e2c07941c175ea2452f1455f570c55510586b387ec2ff2")))
		Expect(bytesA).To(Equal(bytesB))
	})

	//It("repository (from spec v2) with component descriptors and resources stored in distinct directories", func() {
	//	localrootfs.Set(env.OCMContext(), fs)
	//	spec := Must(local.NewRepositorySpecV2(fs, filepath.Join(DISTINCT_REPOSITORY, "compdescs"), filepath.Join(DISTINCT_REPOSITORY, "blobs")))
	//	repo := Must(spec.Repository(env.OCMContext(), nil))
	//	defer Close(repo)
	//	cv := Must(repo.LookupComponentVersion(COMPONENT_NAME, COMPONENT_VERSION))
	//	defer Close(cv)
	//	res := Must(cv.GetResourcesByName(RESOURCE_NAME))
	//	acc := Must(res[0].AccessMethod())
	//	defer Close(acc)
	//	bufferA := Must(acc.Get())
	//
	//	bufferB := Must(vfs.ReadFile(fs, filepath.Join(DISTINCT_REPOSITORY, "blobs", "blob1")))
	//	Expect(bufferA).To(Equal(bufferB))
	//})
	//
	//It("repository (from spec v2) with a directory resource", func() {
	//	localrootfs.Set(env.OCMContext(), fs)
	//	spec := Must(local.NewRepositorySpecV2(fs, DIRECTORY_REPOSITORY, DIRECTORY_REPOSITORY))
	//	repo := Must(spec.Repository(env.OCMContext(), nil))
	//	defer Close(repo)
	//	cv := Must(repo.LookupComponentVersion(COMPONENT_NAME, COMPONENT_VERSION))
	//	defer Close(cv)
	//	res := Must(cv.GetResourcesByName(RESOURCE_NAME))
	//	acc := Must(res[0].AccessMethod())
	//	defer Close(acc)
	//	data := Must(acc.Reader())
	//	defer Close(data)
	//
	//	mfs := memoryfs.New()
	//	_, _, err := tarutils.ExtractTarToFsWithInfo(mfs, data)
	//	Expect(err).ToNot(HaveOccurred())
	//	bufferA := Must(vfs.ReadFile(mfs, "testblob"))
	//	bufferB := Must(vfs.ReadFile(fs, filepath.Join(DIRECTORY_REPOSITORY, "blob1", "testblob")))
	//	Expect(bufferA).To(Equal(bufferB))
	//})
	//
	//It("manage seperate attribute contexts", func() {
	//	octx1 := ocm.New(datacontext.MODE_EXTENDED)
	//	octx2 := ocm.New(datacontext.MODE_EXTENDED)
	//
	//	fs2 := memoryfs.New()
	//	localrootfs.Set(octx1, fs)
	//	localrootfs.Set(octx2, fs2)
	//	Expect(localrootfs.Get(octx1)).To(BeIdenticalTo(fs))
	//	Expect(localrootfs.Get(octx2)).To(BeIdenticalTo(fs2))
	//})
})
