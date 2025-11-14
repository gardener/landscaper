// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
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

	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/componentarchive/resources"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/componentarchive"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/template"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/utils"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Resources Test Suite")
}

var _ = ginkgo.Describe("Add", func() {

	var testdataFs vfs.FileSystem

	ginkgo.BeforeEach(func() {
		fs, err := projectionfs.New(osfs.New(), "./testdata")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), fs)
	})

	ginkgo.It("should add a resource defined by a file", func() {
		opts := &resources.Options{
			BuilderOptions:      componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
			ResourceObjectPaths: []string{"./resources/00-res.yaml"},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("ubuntu"),
			"Version":       Equal("v0.0.1"),
			"Type":          Equal("ociImage"),
			"ExtraIdentity": HaveLen(1),
		}))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ResourceRelation("external")),
		}))
		Expect(cd.Resources[0].Access.Object).To(HaveKeyWithValue("type", "ociRegistry"))
		Expect(cd.Resources[0].Access.Object).To(HaveKeyWithValue("imageReference", "ubuntu:18.0"))
	})

	ginkgo.It("should add a resource defined arguments", func() {
		opts := &resources.Options{}
		Expect(opts.Complete([]string{"./00-component", "./resources/00-res.yaml"})).To(Succeed())

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("ubuntu"),
			"Version":       Equal("v0.0.1"),
			"Type":          Equal("ociImage"),
			"ExtraIdentity": HaveLen(1),
		}))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ResourceRelation("external")),
		}))
		Expect(cd.Resources[0].Access.Object).To(HaveKeyWithValue("type", "ociRegistry"))
		Expect(cd.Resources[0].Access.Object).To(HaveKeyWithValue("imageReference", "ubuntu:18.0"))
	})

	ginkgo.It("should add a resource defined by the deprecated -r option", func() {
		opts := &resources.Options{
			ResourceObjectPath: "./resources/00-res.yaml",
		}
		Expect(opts.Complete([]string{"./00-component"})).To(Succeed())

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("ubuntu"),
			"Version":       Equal("v0.0.1"),
			"Type":          Equal("ociImage"),
			"ExtraIdentity": HaveLen(1),
		}))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ResourceRelation("external")),
		}))
		Expect(cd.Resources[0].Access.Object).To(HaveKeyWithValue("type", "ociRegistry"))
		Expect(cd.Resources[0].Access.Object).To(HaveKeyWithValue("imageReference", "ubuntu:18.0"))
	})

	ginkgo.It("should add a resource defined by stdin", func() {
		input, err := os.Open("./testdata/resources/00-res.yaml")
		Expect(err).ToNot(HaveOccurred())
		defer input.Close()
		oldstdin := os.Stdin
		defer func() {
			os.Stdin = oldstdin
		}()
		os.Stdin = input

		opts := &resources.Options{
			BuilderOptions:      componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
			ResourceObjectPaths: []string{"-"},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("ubuntu"),
			"Version": Equal("v0.0.1"),
			"Type":    Equal("ociImage"),
		}))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ResourceRelation("external")),
		}))
		Expect(cd.Resources[0].Access.Object).To(HaveKeyWithValue("type", "ociRegistry"))
		Expect(cd.Resources[0].Access.Object).To(HaveKeyWithValue("imageReference", "ubuntu:18.0"))
	})

	ginkgo.It("should add a resource defined by stdin if nothing is defined", func() {
		input, err := os.Open("./testdata/resources/00-res.yaml")
		Expect(err).ToNot(HaveOccurred())
		defer input.Close()
		oldstdin := os.Stdin
		defer func() {
			os.Stdin = oldstdin
		}()
		os.Stdin = input

		opts := &resources.Options{
			BuilderOptions: componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("ubuntu"),
			"Version": Equal("v0.0.1"),
			"Type":    Equal("ociImage"),
		}))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ResourceRelation("external")),
		}))
		Expect(cd.Resources[0].Access.Object).To(HaveKeyWithValue("type", "ociRegistry"))
		Expect(cd.Resources[0].Access.Object).To(HaveKeyWithValue("imageReference", "ubuntu:18.0"))
	})

	ginkgo.It("should automatically set the version for a local resource", func() {
		opts := &resources.Options{
			BuilderOptions:      componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
			ResourceObjectPaths: []string{"./resources/01-local.yaml"},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("testres"),
			"Version": Equal("v0.0.0"),
			"Type":    Equal("mytype"),
		}))
	})

	ginkgo.It("should add multiple resources via multi yaml docs", func() {
		opts := &resources.Options{
			BuilderOptions:      componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
			ResourceObjectPaths: []string{"./resources/02-multidoc.yaml"},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(2))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("ubuntu"),
			"Version": Equal("v0.0.1"),
			"Type":    Equal("ociImage"),
		}))
		Expect(cd.Resources[1].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("testres"),
			"Version": Equal("v0.0.0"),
			"Type":    Equal("mytype"),
		}))
	})

	ginkgo.It("should add multiple resources via resource list", func() {
		opts := &resources.Options{
			BuilderOptions:      componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
			ResourceObjectPaths: []string{"./resources/05-resource-list.yaml"},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(2))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("ubuntu"),
			"Version": Equal("v0.0.1"),
			"Type":    Equal("ociImage"),
		}))
		Expect(cd.Resources[1].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("testres"),
			"Version": Equal("v0.0.0"),
			"Type":    Equal("mytype"),
		}))
	})

	ginkgo.It("should overwrite the version of a already existing resource", func() {
		opts := &resources.Options{
			BuilderOptions:      componentarchive.BuilderOptions{ComponentArchivePath: "./01-component"},
			ResourceObjectPaths: []string{"./resources/03-overwrite.yaml"},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("ubuntu"),
			"Version": Equal("v0.0.2"),
			"Type":    Equal("ociImage"),
		}))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ResourceRelation("external")),
		}))
		Expect(cd.Resources[0].Access.Object).To(HaveKeyWithValue("type", "ociRegistry"))
		Expect(cd.Resources[0].Access.Object).To(HaveKeyWithValue("imageReference", "ubuntu:18.0"))
	})

	ginkgo.It("should throw an error if an invalid resource is defined", func() {
		opts := &resources.Options{
			BuilderOptions:      componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
			ResourceObjectPaths: []string{"./resources/10-res-invalid.yaml"},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(HaveOccurred())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())
		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())
		Expect(cd.Resources).To(HaveLen(0))
	})

	ginkgo.Context("With Input", func() {
		ginkgo.It("should add a resource defined by a file with a jsonfile input", func() {
			opts := &resources.Options{
				BuilderOptions: componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
				// jsonschema example copied from https://json-schema.org/learn/miscellaneous-examples.html
				ResourceObjectPaths: []string{"./resources/20-res-json.yaml"},
			}

			Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

			data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
			Expect(err).ToNot(HaveOccurred())
			cd := &cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, cd)).To(Succeed())

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

			blobs, err := vfs.ReadDir(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.BlobsDirectoryName))
			Expect(err).ToNot(HaveOccurred())
			Expect(blobs).To(HaveLen(1))
		})

		ginkgo.It("should automatically tar a directory input and add it as resource", func() {
			opts := &resources.Options{
				BuilderOptions:      componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
				ResourceObjectPaths: []string{"./resources/20-res-json.yaml"},
			}

			Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

			data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
			Expect(err).ToNot(HaveOccurred())
			cd := &cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, cd)).To(Succeed())

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

			blobs, err := vfs.ReadDir(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.BlobsDirectoryName))
			Expect(err).ToNot(HaveOccurred())
			Expect(blobs).To(HaveLen(1))
		})

		ginkgo.It("should gzip a input blob and add it as resource if the gzip flag is provided", func() {
			opts := &resources.Options{
				BuilderOptions:      componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
				ResourceObjectPaths: []string{"./resources/21-res-dir-zip.yaml"},
			}

			Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

			data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
			Expect(err).ToNot(HaveOccurred())
			cd := &cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, cd)).To(Succeed())

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

			blobs, err := vfs.ReadDir(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.BlobsDirectoryName))
			Expect(err).ToNot(HaveOccurred())
			Expect(blobs).To(HaveLen(1))

			mimetype, err := utils.GetFileType(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.BlobsDirectoryName, blobs[0].Name()))
			Expect(err).ToNot(HaveOccurred())
			Expect(mimetype).To(Equal("application/x-gzip"))
		})

		ginkgo.It("should automatically tar a directory input and add it as resource and include ", func() {
			opts := &resources.Options{
				BuilderOptions:      componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
				ResourceObjectPaths: []string{"./resources/24-res-mul-files-include.yaml"},
			}

			Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

			data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
			Expect(err).ToNot(HaveOccurred())
			cd := &cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, cd)).To(Succeed())

			Expect(cd.Resources).To(HaveLen(1))

			blobs, err := vfs.ReadDir(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.BlobsDirectoryName))
			Expect(err).ToNot(HaveOccurred())
			Expect(blobs).To(HaveLen(1))

			blob, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.BlobsDirectoryName, blobs[0].Name()))
			Expect(err).ToNot(HaveOccurred())
			files, err := untar(blob)
			Expect(err).ToNot(HaveOccurred())

			Expect(files).To(MatchKeys(0, Keys{
				"file1.txt": Equal([]byte("val1")),
				"file2.txt": Equal([]byte("val2")),
			}))
		})

		ginkgo.It("should automatically tar a directory input and add it as resource and exclude ", func() {
			opts := &resources.Options{
				BuilderOptions:      componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
				ResourceObjectPaths: []string{"./resources/24-res-mul-files-exclude.yaml"},
			}

			Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

			data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
			Expect(err).ToNot(HaveOccurred())
			cd := &cdv2.ComponentDescriptor{}
			Expect(codec.Decode(data, cd)).To(Succeed())

			Expect(cd.Resources).To(HaveLen(1))

			blobs, err := vfs.ReadDir(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.BlobsDirectoryName))
			Expect(err).ToNot(HaveOccurred())
			Expect(blobs).To(HaveLen(1))

			blob, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.BlobsDirectoryName, blobs[0].Name()))
			Expect(err).ToNot(HaveOccurred())
			files, err := untar(blob)
			Expect(err).ToNot(HaveOccurred())

			Expect(files).To(MatchKeys(0, Keys{
				"file2.txt": Equal([]byte("val2")),
				"file3":     Equal([]byte("val3")),
			}))
		})

	})

	ginkgo.It("should add a resource defined by a file with a template", func() {
		opts := &resources.Options{
			BuilderOptions: componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
			TemplateOptions: template.Options{
				Vars: map[string]string{
					"MY_VERSION": "v0.0.2",
				},
			},
			ResourceObjectPaths: []string{"./resources/04-res.yaml"},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("ubuntu"),
			"Version":       Equal("v0.0.2"),
			"Type":          Equal("ociImage"),
			"ExtraIdentity": HaveLen(1),
		}))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ResourceRelation("external")),
		}))
		Expect(cd.Resources[0].Access.Object).To(HaveKeyWithValue("type", "ociRegistry"))
		Expect(cd.Resources[0].Access.Object).To(HaveKeyWithValue("imageReference", "ubuntu:v0.0.2"))
	})

	ginkgo.It("should preserve the directory", func() {
		opts := &resources.Options{
			BuilderOptions:      componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
			ResourceObjectPaths: []string{"./resources/23-res-dir.yaml"},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		blobs, err := vfs.ReadDir(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.BlobsDirectoryName))
		Expect(err).ToNot(HaveOccurred())
		Expect(blobs).To(HaveLen(1))

		blobPath := filepath.Join(opts.ComponentArchivePath, ctf.BlobsDirectoryName, blobs[0].Name())
		tarData, err := vfs.ReadFile(testdataFs, blobPath)
		Expect(err).ToNot(HaveOccurred())
		res, err := untar(tarData)
		Expect(err).ToNot(HaveOccurred())

		Expect(res).To(HaveKeyWithValue("22-dir-json", []byte("dir")))
		Expect(res).To(HaveKey("22-dir-json/21-jsonschema.json"))
	})

	ginkgo.It("should follow symlinks in a directory", func() {
		opts := &resources.Options{
			BuilderOptions:      componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
			ResourceObjectPaths: []string{"./resources/25-symlink.yaml"},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		blobs, err := vfs.ReadDir(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.BlobsDirectoryName))
		Expect(err).ToNot(HaveOccurred())
		Expect(blobs).To(HaveLen(1))

		blobPath := filepath.Join(opts.ComponentArchivePath, ctf.BlobsDirectoryName, blobs[0].Name())
		tarData, err := vfs.ReadFile(testdataFs, blobPath)
		Expect(err).ToNot(HaveOccurred())

		res, err := untar(tarData)
		Expect(err).ToNot(HaveOccurred())

		Expect(res).To(HaveKeyWithValue("file1.txt", []byte("val1")))
		Expect(res).To(HaveKeyWithValue("sym", []byte("val3")))
		Expect(res).To(HaveKeyWithValue("symlinkedDir", []byte("dir")))
		Expect(res).To(HaveKeyWithValue("symlinkedDir/file1", []byte("val1")))
	})

	ginkgo.It("should not follow symlinks in a directory", func() {
		opts := &resources.Options{
			BuilderOptions:      componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
			ResourceObjectPaths: []string{"./resources/25-symlink-not.yaml"},
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		blobs, err := vfs.ReadDir(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.BlobsDirectoryName))
		Expect(err).ToNot(HaveOccurred())
		Expect(blobs).To(HaveLen(1))

		blobPath := filepath.Join(opts.ComponentArchivePath, ctf.BlobsDirectoryName, blobs[0].Name())
		tarData, err := vfs.ReadFile(testdataFs, blobPath)
		Expect(err).ToNot(HaveOccurred())

		res, err := untar(tarData)
		Expect(err).ToNot(HaveOccurred())

		Expect(res).To(HaveKeyWithValue("file1.txt", []byte("val1")))
		Expect(res).ToNot(HaveKeyWithValue("sym", []byte("val3")))
		Expect(res).ToNot(HaveKeyWithValue("symlinkedDir", []byte("dir")))
	})

})

func untar(data []byte) (map[string][]byte, error) {
	files := make(map[string][]byte)
	tr := tar.NewReader(bytes.NewBuffer(data))
	for {
		header, err := tr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return files, nil
			}
			return nil, err
		}
		if header.Typeflag == tar.TypeDir {
			files[header.Name] = []byte("dir")
			continue
		}
		var d bytes.Buffer
		if _, err := io.Copy(&d, tr); err != nil {
			return nil, err
		}
		files[header.Name] = d.Bytes()
	}
}
