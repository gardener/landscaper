// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentarchive

import (
	"testing"

	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ComponentArchive Test Suite")
}

var _ = Describe("Archive", func() {

	var testdataFs vfs.FileSystem

	BeforeEach(func() {
		fs, err := projectionfs.New(osfs.New(), "./testdata")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), fs)
	})

	It("should return error for empty component descriptor if name and version not set in options", func() {
		opts := BuilderOptions{ComponentArchivePath: "./00-component"}

		_, err := opts.Build(testdataFs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).Should(ContainSubstring("invalid component descriptor"))
	})

	It("should set the component name and version for empty component descriptor", func() {
		const (
			componentName    = "example.com/component"
			componentVersion = "v0.0.0"
		)

		opts := BuilderOptions{
			ComponentArchivePath: "./00-component",
			Name:                 componentName,
			Version:              componentVersion,
		}

		archive, err := opts.Build(testdataFs)
		Expect(err).ToNot(HaveOccurred())
		Expect(archive.ComponentDescriptor.Name).To(Equal(componentName))
		Expect(archive.ComponentDescriptor.Version).To(Equal(componentVersion))
	})

	It("should return error when trying to overwrite existing component name", func() {
		const (
			componentName    = "example.com/new-component"
			componentVersion = "v0.0.0"
		)

		opts := BuilderOptions{
			ComponentArchivePath: "./01-component",
			Name:                 componentName,
			Version:              componentVersion,
		}

		_, err := opts.Build(testdataFs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).Should(ContainSubstring("unable to overwrite the existing component name: forbidden"))
	})

	It("should return error when trying to overwrite existing component version", func() {
		const (
			componentName    = "example.com/component"
			componentVersion = "v0.0.1"
		)
		opts := BuilderOptions{
			ComponentArchivePath: "./01-component",
			Name:                 componentName,
			Version:              componentVersion,
		}

		_, err := opts.Build(testdataFs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).Should(ContainSubstring("unable to overwrite the existing component version: forbidden"))
	})

	It("should not return error when existing component name and version are equal to opts", func() {
		const (
			componentName    = "example.com/component"
			componentVersion = "v0.0.0"
		)

		opts := BuilderOptions{
			ComponentArchivePath: "./01-component",
			Name:                 componentName,
			Version:              componentVersion,
		}

		_, err := opts.Build(testdataFs)
		Expect(err).ToNot(HaveOccurred())
	})

})
