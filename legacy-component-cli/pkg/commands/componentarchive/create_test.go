// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentarchive_test

import (
	"context"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/codec"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/componentarchive"
)

var _ = ginkgo.Describe("Create", func() {

	var testdataFs vfs.FileSystem

	ginkgo.BeforeEach(func() {
		baseFs, err := projectionfs.New(osfs.New(), "./testdata")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), baseFs)
	})

	ginkgo.Context("Create", func() {
		ginkgo.It("should create a component archive and overwrite with a newer version", func() {
			opts := &componentarchive.CreateOptions{}
			opts.Name = "example.com/component/name"
			opts.Version = "v0.0.1"
			opts.BaseUrl = "example.com/testurl"
			opts.ComponentArchivePath = "./create-test"
			err := testdataFs.Mkdir(opts.ComponentArchivePath, os.ModePerm)
			Expect(err).ToNot(HaveOccurred(), "Should create a directory with name "+opts.ComponentArchivePath)

			Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed(), "Should create a component archive")

			data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
			Expect(err).ToNot(HaveOccurred())

			cd := &cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, cd)).To(Succeed())
			Expect(cd.Name).To(Equal(opts.Name), "component name should be the same")
			Expect(cd.Version).To(Equal(opts.Version), "component version should be the same")

			Expect(cd.RepositoryContexts).To(HaveLen(1), "there should be exactly one repository context")
			repoCtx := cd.RepositoryContexts[0]
			Expect(repoCtx.GetType()).To(Equal(cdv2.OCIRegistryType), "repository context should be OCIRegistryType")
			ociRepoCtx := &cdv2.OCIRegistryRepository{}
			Expect(repoCtx.DecodeInto(ociRepoCtx)).To(Succeed())
			Expect(ociRepoCtx.BaseURL).To(Equal(opts.BaseUrl))
		})

	})

	ginkgo.Context("Overwrite", func() {

		ginkgo.It("should create a component archive and overwrite with a newer version", func() {
			opts := &componentarchive.CreateOptions{}
			opts.Name = "example.com/component/name"
			opts.Version = "v0.0.1"
			opts.BaseUrl = "example.com/testurl"
			opts.ComponentArchivePath = "./overwrite-test"
			err := testdataFs.Mkdir(opts.ComponentArchivePath, os.ModePerm)
			Expect(err).ToNot(HaveOccurred(), "Should create a directory with name "+opts.ComponentArchivePath)

			Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed(), "Should create a component archive")

			data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
			Expect(err).ToNot(HaveOccurred())

			cd := &cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, cd)).To(Succeed())
			Expect(cd.Name).To(Equal(opts.Name), "component name should be the same")
			Expect(cd.Version).To(Equal(opts.Version), "component version should be the same")

			Expect(cd.RepositoryContexts).To(HaveLen(1), "there should be exactly one repository context")
			repoCtx := cd.RepositoryContexts[0]
			Expect(repoCtx.GetType()).To(Equal(cdv2.OCIRegistryType), "repository context should be OCIRegistryType")
			ociRepoCtx := &cdv2.OCIRegistryRepository{}
			Expect(repoCtx.DecodeInto(ociRepoCtx)).To(Succeed())
			Expect(ociRepoCtx.BaseURL).To(Equal(opts.BaseUrl))

			// check overwrite
			opts.Version = "v0.0.2"
			Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(HaveOccurred(), "Should not overwrite existing component as Overwrite=false")

			opts.Overwrite = true
			Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed(), "Should overwrite existing component as Overwrite=true")

			data, err = vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
			Expect(err).ToNot(HaveOccurred())

			cd = &cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, cd)).To(Succeed())
			Expect(cd.Name).To(Equal(opts.Name), "component name should be the same")
			Expect(cd.Version).To(Equal(opts.Version), "component version should be the same")

			Expect(cd.RepositoryContexts).To(HaveLen(1), "there should be exactly one repository context")
			repoCtx = cd.RepositoryContexts[0]
			Expect(repoCtx.GetType()).To(Equal(cdv2.OCIRegistryType), "repository context should be OCIRegistryType")
			ociRepoCtx = &cdv2.OCIRegistryRepository{}
			Expect(repoCtx.DecodeInto(ociRepoCtx)).To(Succeed())
			Expect(ociRepoCtx.BaseURL).To(Equal(opts.BaseUrl))
		})
	})

})
