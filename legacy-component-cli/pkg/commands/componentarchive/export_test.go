// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentarchive_test

import (
	"context"
	"path/filepath"

	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/componentarchive"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/utils"
)

var _ = ginkgo.Describe("Export", func() {

	var testdataFs vfs.FileSystem

	ginkgo.BeforeEach(func() {
		baseFs, err := projectionfs.New(osfs.New(), "./testdata")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), baseFs)
	})

	ginkgo.Context("From Filesystem", func() {

		ginkgo.It("should export a component archive from filesystem as tar file", func() {
			opts := &componentarchive.ExportOptions{
				ComponentArchivePath: "00-ca",
				OutputPath:           "ca.tar",
				OutputFormat:         ctf.ArchiveFormatTar,
			}

			Expect(opts.Run(context.TODO(), testdataFs)).To(Succeed())
			mediatype, err := utils.GetFileType(testdataFs, "ca.tar")
			Expect(err).ToNot(HaveOccurred())
			Expect(mediatype).ToNot(ContainSubstring("gzip"))
		})

		ginkgo.It("should export a component archive from filesystem as tar file", func() {
			opts := &componentarchive.ExportOptions{
				ComponentArchivePath: "00-ca",
				OutputPath:           "ca.tar.gz",
				OutputFormat:         ctf.ArchiveFormatTarGzip,
			}

			Expect(opts.Run(context.TODO(), testdataFs)).To(Succeed())
			mediatype, err := utils.GetFileType(testdataFs, "ca.tar.gz")
			Expect(err).ToNot(HaveOccurred())
			Expect(mediatype).To(Equal("application/x-gzip"))
		})

	})

	ginkgo.Context("From tar", func() {

		ginkgo.It("should export a component archive as tar file to filesystem", func() {
			opts := &componentarchive.ExportOptions{
				ComponentArchivePath: "00-ca",
				OutputPath:           "ca.tar",
				OutputFormat:         ctf.ArchiveFormatTar,
			}

			Expect(opts.Run(context.TODO(), testdataFs)).To(Succeed())

			opts = &componentarchive.ExportOptions{
				ComponentArchivePath: "ca.tar",
				OutputPath:           "ca",
				OutputFormat:         ctf.ArchiveFormatFilesystem,
			}

			Expect(opts.Run(context.TODO(), testdataFs)).To(Succeed())
			outputfileinfo, err := testdataFs.Stat("ca")
			Expect(err).ToNot(HaveOccurred())
			Expect(outputfileinfo.IsDir()).To(BeTrue(), "output filepath should be a directory")

			outputfileinfo, err = testdataFs.Stat(filepath.Join("ca", ctf.ComponentDescriptorFileName))
			Expect(err).ToNot(HaveOccurred())
			Expect(outputfileinfo.IsDir()).To(BeFalse(), "output filepath should be a directory")
		})

	})

})
