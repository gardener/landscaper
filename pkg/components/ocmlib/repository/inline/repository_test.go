// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package inline_test

import (
	"github.com/mandelsoft/filepath/pkg/filepath"
	. "github.com/mandelsoft/goutils/testutils"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"ocm.software/ocm/api/datacontext/attrs/vfsattr"
	. "ocm.software/ocm/api/helper/builder"
	tenv "ocm.software/ocm/api/helper/env"
	"ocm.software/ocm/api/ocm/compdesc"
	"ocm.software/ocm/api/ocm/selectors/rscsel"
	"ocm.software/ocm/api/utils/runtime"
	"ocm.software/ocm/api/utils/tarutils"

	"github.com/gardener/landscaper/pkg/components/ocmlib/repository"
	"github.com/gardener/landscaper/pkg/components/ocmlib/repository/inline"
)

const (
	// Component Names
	COMPONENT_NAME    = "example.com/root"
	COMPONENT_VERSION = "1.0.0"
	RESOURCE_NAME     = "test"

	DISTINCT_REPOSITORY  = "testdata/distinct"
	DIRECTORY_REPOSITORY = "testdata/directory"
)

var _ = Describe("ocm-lib based landscaper local repository", func() {
	var env *Builder

	BeforeEach(func() {
		env = NewBuilder(tenv.NewEnvironment(tenv.TestData()))
	})

	AfterEach(func() {
		_ = env.Cleanup()
	})

	It("repository spec with inline component descriptors and local resources", func() {
		specdata := Must(vfs.ReadFile(osfs.New(), filepath.Join("testdata", "spec-with-local-resource.yaml")))
		vfsattr.Set(env.OCMContext(), env)
		spec := Must(env.OCMContext().RepositorySpecForConfig(specdata, runtime.DefaultYAMLEncoding))
		repo := Must(spec.Repository(env.OCMContext(), nil))
		defer repo.Close()
		cv := Must(repo.LookupComponentVersion(COMPONENT_NAME, COMPONENT_VERSION))
		defer cv.Close()
		ref := Must(cv.GetReferenceByIndex(0))
		refcv := Must(repo.LookupComponentVersion(ref.ComponentName, ref.Version))
		defer refcv.Close()
		res := Must(cv.SelectResources(rscsel.Name(RESOURCE_NAME)))
		acc := Must(res[0].AccessMethod())
		defer acc.Close()
		data := Must(acc.Get())
		Expect(string(data)).To(Equal("test"))
	})

	It("repository spec with inline component descriptors and inline resources", func() {
		specdata := Must(vfs.ReadFile(osfs.New(), filepath.Join("testdata", "spec-with-inline-resource.yaml")))

		spec := Must(env.OCMContext().RepositorySpecForConfig(specdata, runtime.DefaultYAMLEncoding))
		repo := Must(spec.Repository(env.OCMContext(), nil))
		defer repo.Close()
		cv := Must(repo.LookupComponentVersion(COMPONENT_NAME, COMPONENT_VERSION))
		defer cv.Close()
		ref := Must(cv.GetReferenceByIndex(0))
		refcv := Must(repo.LookupComponentVersion(ref.ComponentName, ref.Version))
		defer refcv.Close()
		res := Must(cv.SelectResources(rscsel.Name(RESOURCE_NAME)))
		acc := Must(res[0].AccessMethod())
		defer acc.Close()
		data := Must(acc.Get())
		Expect(string(data)).To(Equal("test"))
	})

	It("support for legacy inline component descriptors and resources", func() {
		inline1 := Must(vfs.ReadFile(osfs.New(), filepath.Join("testdata", "legacy", "component-descriptor1.yaml")))
		inline2 := Must(vfs.ReadFile(osfs.New(), filepath.Join("testdata", "legacy", "component-descriptor2.yaml")))
		resource1 := Must(vfs.ReadFile(osfs.New(), filepath.Join("testdata", "legacy", "resource.yaml")))

		var list []*compdesc.ComponentDescriptor
		list = append(list, Must(compdesc.Decode(inline1)))
		list = append(list, Must(compdesc.Decode(inline2)))

		memfs := memoryfs.New()
		r1 := Must(memfs.Create("blob1"))
		Must(r1.Write(resource1))
		MustBeSuccessful(r1.Close())

		repo := Must(repository.NewRepository(env.OCMContext(), repository.NewMemoryCompDescProvider(list), memfs))
		defer repo.Close()
		cv := Must(repo.LookupComponentVersion(COMPONENT_NAME, COMPONENT_VERSION))
		defer cv.Close()
		ref := Must(cv.GetReferenceByIndex(0))
		refcv := Must(repo.LookupComponentVersion(ref.ComponentName, ref.Version))
		defer refcv.Close()
		res := Must(cv.SelectResources(rscsel.Name(RESOURCE_NAME)))
		acc := Must(res[0].AccessMethod())
		defer acc.Close()
		data := Must(acc.Get())
		Expect(string(data)).To(Equal("test"))
	})

	It("repository with component descriptors and resources stored in distinct directories", func() {
		spec := Must(inline.NewRepositorySpecV1(env, filepath.Join(DISTINCT_REPOSITORY, "compdescs"), nil, filepath.Join(DISTINCT_REPOSITORY, "blobs")))
		repo := Must(spec.Repository(env.OCMContext(), nil))
		defer repo.Close()
		cv := Must(repo.LookupComponentVersion(COMPONENT_NAME, COMPONENT_VERSION))
		defer cv.Close()
		res := Must(cv.SelectResources(rscsel.Name(RESOURCE_NAME)))
		acc := Must(res[0].AccessMethod())
		defer acc.Close()
		bufferA := Must(acc.Get())

		bufferB := Must(vfs.ReadFile(env, filepath.Join(DISTINCT_REPOSITORY, "blobs", "blob1")))
		Expect(bufferA).To(Equal(bufferB))
	})

	It("repository with a directory resource", func() {
		spec := Must(inline.NewRepositorySpecV1(env, DIRECTORY_REPOSITORY, nil, DIRECTORY_REPOSITORY))
		repo := Must(spec.Repository(env.OCMContext(), nil))
		defer repo.Close()
		cv := Must(repo.LookupComponentVersion(COMPONENT_NAME, COMPONENT_VERSION))
		defer cv.Close()
		res := Must(cv.SelectResources(rscsel.Name(RESOURCE_NAME)))
		acc := Must(res[0].AccessMethod())
		defer acc.Close()
		data := Must(acc.Reader())
		defer data.Close()

		mfs := memoryfs.New()
		_, _, err := tarutils.ExtractTarToFsWithInfo(mfs, data)
		Expect(err).ToNot(HaveOccurred())
		bufferA := Must(vfs.ReadFile(mfs, "testblob"))
		bufferB := Must(vfs.ReadFile(env, filepath.Join(DIRECTORY_REPOSITORY, "blob1", "testblob")))
		Expect(bufferA).To(Equal(bufferB))
	})

})
