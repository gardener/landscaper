// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentarchive_test

import (
	"bytes"
	"context"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/componentarchive"
)

var _ = Describe("Add", func() {

	var testdataFs vfs.FileSystem

	BeforeEach(func() {
		baseFs, err := projectionfs.New(osfs.New(), "./")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), baseFs)
	})

	It("should add a component descriptor from file to the ctf archive", func() {
		baseFs, err := projectionfs.New(osfs.New(), "./testdata")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), baseFs)

		ctx := context.Background()
		defer ctx.Done()
		opts := &componentarchive.ComponentArchiveOptions{
			CTFPath:       "/component.ctf",
			ArchiveFormat: ctf.ArchiveFormatTar,
		}
		opts.ComponentArchivePath = "./00-ca"

		Expect(opts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())

		ctfArchive, err := ctf.NewCTF(testdataFs, opts.CTFPath)
		Expect(err).ToNot(HaveOccurred())
		ca := getComponentDescriptorFromCTF(ctfArchive)
		Expect(ca.ComponentDescriptor.Name).To(Equal("example.com/component"))
	})

	It("should add a component descriptor with a resource from a file to the ctf archive", func() {
		ctx := context.Background()
		defer ctx.Done()
		opts := &componentarchive.ComponentArchiveOptions{
			CTFPath:        "/component.ctf",
			ArchiveFormat:  ctf.ArchiveFormatTar,
			ResourcesPaths: []string{"./resources/testdata/resources/21-res-dir.yaml"},
		}
		opts.ComponentArchivePath = "./testdata/00-ca"

		Expect(opts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())

		ctfArchive, err := ctf.NewCTF(testdataFs, opts.CTFPath)
		Expect(err).ToNot(HaveOccurred())
		ca := getComponentDescriptorFromCTF(ctfArchive)
		Expect(ca.ComponentDescriptor.Name).To(Equal("example.com/component"))

		cd := ca.ComponentDescriptor
		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("myconfig"),
			"Version": Equal("v0.0.1"),
			"Type":    Equal("jsonschema"),
		}))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ResourceRelation("external")),
		}))
		Expect(cd.Resources[0].Access.Object).To(HaveKeyWithValue("type", cdv2.LocalFilesystemBlobType))
		Expect(cd.Resources[0].Access.Object).To(HaveKeyWithValue("filename", BeAssignableToTypeOf("")))

		var data bytes.Buffer
		info, err := ca.Resolve(ctx, cd.Resources[0], &data)
		Expect(err).ToNot(HaveOccurred())
		Expect(info.MediaType).To(Equal("application/x-tar"))
		Expect(data.Len()).To(BeNumerically(">", 0))
	})

})

func getComponentDescriptorFromCTF(ctfArchive *ctf.CTF) *ctf.ComponentArchive {
	var archive *ctf.ComponentArchive
	Expect(ctfArchive.Walk(func(ca *ctf.ComponentArchive) error {
		archive = ca
		return nil
	})).To(Succeed())
	return archive
}
