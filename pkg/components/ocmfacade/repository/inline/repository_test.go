package inline_test

import (
	"github.com/gardener/landscaper/pkg/components/ocmfacade/repository/attrs/blobvfs"
	"github.com/mandelsoft/filepath/pkg/filepath"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/open-component-model/ocm/pkg/env/builder"
	. "github.com/open-component-model/ocm/pkg/testutils"

	tenv "github.com/open-component-model/ocm/pkg/env"
	"github.com/open-component-model/ocm/pkg/runtime"
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
	specdata1 := Must(vfs.ReadFile(osfs.New(), filepath.Join("testdata", "spec-with-local-resource.yaml")))
	specdata2 := Must(vfs.ReadFile(osfs.New(), filepath.Join("testdata", "spec-with-inline-resource.yaml")))

	BeforeEach(func() {
		env = NewBuilder(tenv.NewEnvironment())
		fs = Must(projectionfs.New(osfs.New(), "../"))
	})

	AfterEach(func() {
		env.Cleanup()
	})

	It("repository spec with inline component descriptors and local resources", func() {
		blobvfs.Set(env.OCMContext(), fs)
		spec := Must(env.OCMContext().RepositorySpecForConfig(specdata1, runtime.DefaultYAMLEncoding))
		repo := Must(spec.Repository(env.OCMContext(), nil))
		defer Close(repo)
		cv := Must(repo.LookupComponentVersion(COMPONENT_NAME, COMPONENT_VERSION))
		defer Close(cv)
		ref := Must(cv.GetReferenceByIndex(0))
		refcv := Must(repo.LookupComponentVersion(ref.ComponentName, ref.Version))
		defer Close(refcv)
		res := Must(cv.GetResourcesByName(RESOURCE_NAME))
		acc := Must(res[0].AccessMethod())
		defer Close(acc)
		data := Must(acc.Get())
		_ = data
	})

	FIt("repository spec with inline component descriptors and inline resources", func() {
		spec := Must(env.OCMContext().RepositorySpecForConfig(specdata2, runtime.DefaultYAMLEncoding))
		repo := Must(spec.Repository(env.OCMContext(), nil))
		defer Close(repo)
		cv := Must(repo.LookupComponentVersion(COMPONENT_NAME, COMPONENT_VERSION))
		defer Close(cv)
		ref := Must(cv.GetReferenceByIndex(0))
		refcv := Must(repo.LookupComponentVersion(ref.ComponentName, ref.Version))
		defer Close(refcv)
		res := Must(cv.GetResourcesByName(RESOURCE_NAME))
		acc := Must(res[0].AccessMethod())
		defer Close(acc)
		data := Must(acc.Get())
		_ = data
	})

})
