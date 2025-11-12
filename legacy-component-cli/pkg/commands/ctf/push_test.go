// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ctf_test

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/landscaper/legacy-component-spec/bindings-go/oci"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient/options"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/componentarchive"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/componentarchive/resources"
	cmd "github.com/gardener/landscaper/legacy-component-cli/pkg/commands/ctf"
)

var _ = Describe("Add", func() {

	var testdataFs vfs.FileSystem

	BeforeEach(func() {
		baseFs, err := projectionfs.New(osfs.New(), "./testdata")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), baseFs)
	})

	It("should push a component archive with a local file", func() {
		baseFs, err := projectionfs.New(osfs.New(), "../componentarchive")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), baseFs)
		ctx := context.Background()

		caOpts := &componentarchive.ComponentArchiveOptions{
			CTFPath:        "/component.ctf",
			ArchiveFormat:  ctf.ArchiveFormatTar,
			ResourcesPaths: []string{"./resources/testdata/resources/21-res-dir.yaml"},
		}
		caOpts.ComponentArchivePath = "./testdata/00-ca"

		Expect(caOpts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())

		_, err = ctf.NewCTF(testdataFs, caOpts.CTFPath)
		Expect(err).ToNot(HaveOccurred())

		cf, err := testenv.GetConfigFileBytes()
		Expect(err).ToNot(HaveOccurred())
		Expect(vfs.WriteFile(testdataFs, "/auth.json", cf, os.ModePerm))

		opts := cmd.PushOptions{
			CTFPath: "/component.ctf",
			BaseUrl: testenv.Addr + "/test",
			OciOptions: options.Options{
				AllowPlainHttp:     false,
				RegistryConfigPath: "/auth.json",
			},
		}
		Expect(opts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())

		repos, err := client.ListRepositories(ctx, testenv.Addr+"/test")
		Expect(err).ToNot(HaveOccurred())

		expectedRef := testenv.Addr + "/test/component-descriptors/example.com/component"
		Expect(repos).To(ContainElement(Equal(expectedRef)))

		manifest, err := client.GetManifest(ctx, expectedRef+":v0.0.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(manifest.Layers).To(HaveLen(2))
		Expect(manifest.Layers[0].MediaType).To(Equal(cdoci.ComponentDescriptorTarMimeType),
			"Expect that the first layer contains the component descriptor")
		Expect(manifest.Layers[1].MediaType).To(Equal("application/x-tar"),
			"Expect that the second layer contains the local blob")
	})

	It("should throw an error if a local resource does not exist", func() {
		baseFs, err := projectionfs.New(osfs.New(), "../componentarchive")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), baseFs)
		ctx := context.Background()

		resOpts := &resources.Options{
			ResourceObjectPaths: []string{"./resources/testdata/resources/21-res-dir.yaml"},
		}
		resOpts.ComponentArchivePath = "./testdata/00-ca"
		Expect(resOpts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())

		// delete blobs
		files, err := vfs.ReadDir(testdataFs, "./testdata/00-ca/blobs")
		Expect(err).ToNot(HaveOccurred())
		Expect(files).To(HaveLen(1))
		Expect(testdataFs.RemoveAll("./testdata/00-ca/blobs")).To(Succeed())
		_, err = vfs.ReadDir(testdataFs, "./testdata/00-ca/blobs")
		Expect(os.IsNotExist(err)).To(BeTrue(), "folder should not exist anymore")

		caOpts := &componentarchive.ComponentArchiveOptions{
			CTFPath:       "/component.ctf",
			ArchiveFormat: ctf.ArchiveFormatTar,
		}
		caOpts.ComponentArchivePath = "./testdata/00-ca"
		Expect(caOpts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())

		_, err = ctf.NewCTF(testdataFs, caOpts.CTFPath)
		Expect(err).ToNot(HaveOccurred())

		cf, err := testenv.GetConfigFileBytes()
		Expect(err).ToNot(HaveOccurred())
		Expect(vfs.WriteFile(testdataFs, "/auth.json", cf, os.ModePerm))

		opts := cmd.PushOptions{
			CTFPath: "/component.ctf",
			BaseUrl: testenv.Addr + "/test",
			OciOptions: options.Options{
				AllowPlainHttp:     false,
				RegistryConfigPath: "/auth.json",
			},
		}
		Expect(opts.Run(ctx, logr.Discard(), testdataFs)).To(HaveOccurred())
	})

})
