// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ctf_test

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"

	cmd "github.com/gardener/landscaper/legacy-component-cli/pkg/commands/ctf"
)

var _ = Describe("Add", func() {

	var testdataFs vfs.FileSystem

	BeforeEach(func() {
		baseFs, err := projectionfs.New(osfs.New(), "./testdata")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), baseFs)
	})

	It("should add a component descriptor from file to the ctf archive", func() {
		ctx := context.Background()
		defer ctx.Done()
		opts := cmd.AddOptions{
			CTFPath:           "/component.ctf",
			ArchiveFormat:     ctf.ArchiveFormatTar,
			ComponentArchives: []string{"./00-ca"},
		}

		Expect(opts.Run(ctx, logr.Discard(), testdataFs)).To(Succeed())

		ctfArchive, err := ctf.NewCTF(testdataFs, opts.CTFPath)
		Expect(err).ToNot(HaveOccurred())
		err = ctfArchive.Walk(func(ca *ctf.ComponentArchive) error {
			Expect(ca.ComponentDescriptor.Name).To(Equal("example.com/component"))
			return nil
		})
		Expect(err).ToNot(HaveOccurred())
	})

})
