// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package sources_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/codec"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/componentarchive/sources"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/componentarchive"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/template"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Sources Test Suite")
}

var _ = ginkgo.Describe("Add", func() {

	var testdataFs vfs.FileSystem

	ginkgo.BeforeEach(func() {
		fs, err := projectionfs.New(osfs.New(), "./testdata")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), fs)
	})

	ginkgo.It("should add a source defined by a file", func() {
		opts := &sources.Options{
			BuilderOptions:    componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
			SourceObjectPaths: []string{"./resources/00-src.yaml"},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Sources).To(HaveLen(1))
		Expect(cd.Sources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("repo"),
			"Version": Equal("v0.0.1"),
			"Type":    Equal("git"),
		}))
	})

	ginkgo.It("should add a source defined by the arguments", func() {
		opts := &sources.Options{}
		Expect(opts.Complete([]string{"./00-component", "./resources/00-src.yaml"})).To(Succeed())

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Sources).To(HaveLen(1))
		Expect(cd.Sources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("repo"),
			"Version": Equal("v0.0.1"),
			"Type":    Equal("git"),
		}))
	})

	ginkgo.It("should add a source file defined by the deprecated -r flag", func() {
		opts := &sources.Options{
			SourceObjectPath: "./resources/00-src.yaml",
		}
		Expect(opts.Complete([]string{"./00-component"})).To(Succeed())

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Sources).To(HaveLen(1))
		Expect(cd.Sources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("repo"),
			"Version": Equal("v0.0.1"),
			"Type":    Equal("git"),
		}))
	})

	ginkgo.It("should add a source defined by stdin when the resource path is '-'", func() {
		input, err := os.Open("./testdata/resources/00-src.yaml")
		Expect(err).ToNot(HaveOccurred())
		defer input.Close()
		oldstdin := os.Stdin
		defer func() {
			os.Stdin = oldstdin
		}()
		os.Stdin = input

		opts := &sources.Options{
			BuilderOptions:    componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
			SourceObjectPaths: []string{"-"},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Sources).To(HaveLen(1))
		Expect(cd.Sources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("repo"),
			"Version": Equal("v0.0.1"),
			"Type":    Equal("git"),
		}))
	})

	ginkgo.It("should add a source defined by stdin when no other inputs are defined.", func() {
		input, err := os.Open("./testdata/resources/00-src.yaml")
		Expect(err).ToNot(HaveOccurred())
		defer input.Close()
		oldstdin := os.Stdin
		defer func() {
			os.Stdin = oldstdin
		}()
		os.Stdin = input

		opts := &sources.Options{
			BuilderOptions: componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Sources).To(HaveLen(1))
		Expect(cd.Sources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("repo"),
			"Version": Equal("v0.0.1"),
			"Type":    Equal("git"),
		}))
	})

	ginkgo.It("should add multiple sources defined by a multi doc file", func() {

		opts := &sources.Options{
			BuilderOptions:    componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
			SourceObjectPaths: []string{"./resources/01-multi-doc.yaml"},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Sources).To(HaveLen(2))
		Expect(cd.Sources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("repo"),
			"Version": Equal("v0.0.1"),
			"Type":    Equal("git"),
		}))
		Expect(cd.Sources[1].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("base-repo"),
			"Version": Equal("v18.4.0"),
			"Type":    Equal("git"),
		}))
	})

	ginkgo.It("should throw an error if an invalid source is defined", func() {
		opts := &sources.Options{
			BuilderOptions:    componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
			SourceObjectPaths: []string{"./resources/10-invalid.yaml"},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(HaveOccurred())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())
		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())
		Expect(cd.Sources).To(HaveLen(0))
	})

	ginkgo.It("should overwrite the version of a already existing source", func() {

		opts := &sources.Options{
			BuilderOptions:    componentarchive.BuilderOptions{ComponentArchivePath: "./01-component"},
			SourceObjectPaths: []string{"./resources/02-overwrite.yaml"},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Sources).To(HaveLen(1))
		Expect(cd.Sources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("repo"),
			"Version": Equal("v0.0.2"),
			"Type":    Equal("git"),
		}))
	})

	ginkgo.It("should add a templated source defined by a file", func() {
		opts := &sources.Options{
			BuilderOptions: componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
			TemplateOptions: template.Options{
				Vars: map[string]string{
					"MY_VERSION": "v0.0.2",
				},
			},
			SourceObjectPaths: []string{"./resources/03-src.yaml"},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Sources).To(HaveLen(1))
		Expect(cd.Sources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("repo"),
			"Version": Equal("v0.0.2"),
			"Type":    Equal("git"),
		}))
	})

})
