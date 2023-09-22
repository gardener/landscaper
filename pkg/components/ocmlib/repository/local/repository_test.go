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

	"github.com/gardener/landscaper/pkg/components/ocmlib/repository"
	"github.com/gardener/landscaper/pkg/components/ocmlib/repository/local"

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
})
