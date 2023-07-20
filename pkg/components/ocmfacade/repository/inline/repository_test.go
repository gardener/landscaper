package inline_test

import (
	"github.com/gardener/landscaper/pkg/components/ocmfacade/repository/attrs/blobvfs"
	"github.com/gardener/landscaper/pkg/components/ocmfacade/repository/attrs/compvfs"
	"github.com/gardener/landscaper/pkg/components/ocmfacade/repository/inline"
	"github.com/mandelsoft/filepath/pkg/filepath"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/open-component-model/ocm/pkg/env/builder"
	. "github.com/open-component-model/ocm/pkg/testutils"

	tenv "github.com/open-component-model/ocm/pkg/env"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	// Component Names
	COMPONENT_NAME    = "example.com/root"
	COMPONENT_VERSION = "1.0.0"
	RESOURCE_NAME     = "test"
)

var _ = Describe("ocm-lib based landscaper local repository", func() {
	var env *Builder

	BeforeEach(func() {
		env = NewBuilder(tenv.NewEnvironment())
	})

	AfterEach(func() {
		env.Cleanup()
	})

	It("repository spec with inline component descriptors and local resources", func() {
		fs := Must(projectionfs.New(osfs.New(), "../"))
		specdata := Must(vfs.ReadFile(osfs.New(), filepath.Join("testdata", "spec-with-local-resource.yaml")))

		blobvfs.Set(env.OCMContext(), fs)
		spec := Must(env.OCMContext().RepositorySpecForConfig(specdata, runtime.DefaultYAMLEncoding))
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
		Expect(string(data)).To(Equal("test"))
	})

	It("repository spec with inline component descriptors and inline resources", func() {
		specdata := Must(vfs.ReadFile(osfs.New(), filepath.Join("testdata", "spec-with-inline-resource.yaml")))

		spec := Must(env.OCMContext().RepositorySpecForConfig(specdata, runtime.DefaultYAMLEncoding))
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
		Expect(string(data)).To(Equal("test"))
	})

	It("support for legacy inline component descriptors and resources", func() {
		inline1 := Must(vfs.ReadFile(osfs.New(), filepath.Join("testdata", "legacy", "component-descriptor1.yaml")))
		inline2 := Must(vfs.ReadFile(osfs.New(), filepath.Join("testdata", "legacy", "component-descriptor2.yaml")))
		resource1 := Must(vfs.ReadFile(osfs.New(), filepath.Join("testdata", "legacy", "resource.yaml")))

		memfs := memoryfs.New()

		// you would also do something like this
		Expect(memfs.MkdirAll("component-descriptors", 0o777)).To(Succeed())
		Expect(memfs.MkdirAll("artifacts", 0o777)).To(Succeed())
		f1 := Must(memfs.Create(filepath.Join("component-descriptors", "component-descriptor1.yaml")))
		Must(f1.Write(inline1))
		defer Close(f1)
		f2 := Must(memfs.Create(filepath.Join("component-descriptors", "component-descriptor2.yaml")))
		Must(f2.Write(inline2))
		defer Close(f2)
		f3 := Must(memfs.Create(filepath.Join("artifacts", "blob1")))
		Must(f3.Write(resource1))
		defer Close(f3)
		compvfs.Set(env.OCMContext(), memfs)
		blobvfs.Set(env.OCMContext(), memfs)

		spec := Must(inline.NewRepositorySpecV1(nil, "component-descriptors", nil, "artifacts"))
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
		Expect(string(data)).To(Equal("test"))
	})

})
