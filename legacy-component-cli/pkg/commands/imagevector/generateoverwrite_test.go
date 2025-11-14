// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imagevector_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
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
	. "github.com/onsi/gomega/gstruct"
	"sigs.k8s.io/yaml"

	iv "github.com/gardener/landscaper/legacy-image-vector/pkg"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/codec"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/constants"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/components"

	ivcmd "github.com/gardener/landscaper/legacy-component-cli/pkg/commands/imagevector"
)

var _ = ginkgo.Describe("GenerateOverwrite", func() {

	var testdataFs vfs.FileSystem

	ginkgo.BeforeEach(func() {
		fs, err := projectionfs.New(osfs.New(), "./testdata")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), fs)
	})

	ginkgo.It("should generate a simple image with tag from a component descriptor", func() {
		imageVector := runGenerateOverwrite(testdataFs, "./01-component/component-descriptor.yaml")

		Expect(imageVector.Images).To(HaveLen(3))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name": Equal("pause-container"),
			"Tag":  PointTo(Equal("3.1")),
		})))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name": Equal("pause-container"),
			"Tag":  PointTo(Equal("sha256:eb9086d472747453ad2d5cfa10f80986d9b0afb9ae9c4256fe2887b029566d06")),
		})))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name": Equal("gardenlet"),
			"Tag":  PointTo(Equal("v0.0.0")),
		})))
	})

	ginkgo.It("should generate a image source with a target version", func() {
		runAdd(testdataFs, "./00-component/component-descriptor.yaml", "./resources/10-targetversion.yaml")
		imageVector := runGenerateOverwrite(testdataFs, "./00-component/component-descriptor.yaml")
		Expect(imageVector.Images).To(HaveLen(1))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("metrics-server"),
			"Tag":           PointTo(Equal("v0.4.1")),
			"TargetVersion": PointTo(Equal(">= 1.11")),
		})))
	})

	ginkgo.It("should generate image sources from generic images", func() {
		addOpts := &ivcmd.AddOptions{
			ParseImageOptions: iv.ParseImageOptions{
				GenericDependencies: []string{
					"hyperkube",
				},
			},
		}
		runAdd(testdataFs, "./00-component/component-descriptor.yaml", "./resources/30-generic.yaml", addOpts)

		getOpts := &ivcmd.GenerateOverwriteOptions{}
		getOpts.AdditionalComponentsRefOrPath = []string{"./04-generic-images/component-descriptor.yaml"}
		imageVector := runGenerateOverwrite(testdataFs, "./00-component/component-descriptor.yaml", getOpts)
		Expect(imageVector.Images).To(HaveLen(3))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("hyperkube"),
			"Repository":    Equal("eu.gcr.io/gardener-project/hyperkube"),
			"Tag":           PointTo(Equal("v1.19.2")),
			"TargetVersion": PointTo(Equal("1.19.2")),
		})))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("hyperkube"),
			"Repository":    Equal("k8s.gcr.io/hyperkube"),
			"Tag":           PointTo(Equal("v1.18.6")),
			"TargetVersion": PointTo(Equal("1.18.6")),
		})))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("hyperkube"),
			"Repository":    Equal("k8s.gcr.io/hyperkube"),
			"Tag":           PointTo(Equal("v1.17.10")),
			"TargetVersion": PointTo(Equal("1.17.10")),
		})))
	})

	ginkgo.Context("Integration", func() {

		ginkgo.It("should generate image sources from a gardener component descriptor ", func() {
			res, err := http.Get("https://raw.githubusercontent.com/gardener/gardener/v1.25.1/charts/images.yaml")
			Expect(err).ToNot(HaveOccurred())
			var gardenerImageVectorBytes bytes.Buffer
			_, err = io.Copy(&gardenerImageVectorBytes, res.Body)
			Expect(err).ToNot(HaveOccurred())
			defer res.Body.Close()

			gardenerImageVector := iv.ImageVector{}
			Expect(yaml.Unmarshal(gardenerImageVectorBytes.Bytes(), &gardenerImageVector))

			getOpts := &ivcmd.GenerateOverwriteOptions{}
			getOpts.BaseURL = "eu.gcr.io/gardener-project/development"
			getOpts.ComponentRefOrPath = "github.com/gardener/gardener:v1.25.1"
			getOpts.AdditionalComponentsRefOrPath = []string{
				"06-kubernetes-versions/component-descriptor.yaml",
			}
			getOpts.ImageVectorPath = "./out/iv.yaml"
			Expect(getOpts.Complete(nil)).To(Succeed())
			Expect(getOpts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

			data, err := vfs.ReadFile(testdataFs, getOpts.ImageVectorPath)
			Expect(err).ToNot(HaveOccurred())

			imageVector := &iv.ImageVector{}
			Expect(yaml.Unmarshal(data, imageVector)).To(Succeed())

			// expect all images defined in the gardener image vector to be also part of the generated image vector
			// minus the generic images
			for _, entry := range gardenerImageVector.Images {
				if entry.Tag == nil {
					continue
				}
				fields := Fields{
					"Name": Equal(entry.Name),
					"Tag":  PointTo(Equal(*entry.Tag)),
				}
				if entry.TargetVersion != nil {
					fields["TargetVersion"] = PointTo(Equal(*entry.TargetVersion))
				}
				Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, fields)))
			}
		})

	})

})

func runGenerateOverwrite(fs vfs.FileSystem, caPath string, getOpts ...*ivcmd.GenerateOverwriteOptions) *iv.ImageVector {
	Expect(len(getOpts) <= 1).To(BeTrue())
	opts := &ivcmd.GenerateOverwriteOptions{}
	if len(getOpts) == 1 {
		opts = getOpts[0]
	}
	opts.ComponentRefOrPath = caPath
	opts.ImageVectorPath = "./out/iv.yaml"
	Expect(opts.Complete(nil)).To(Succeed())

	// fake local cache with given component descriptor
	data, err := vfs.ReadFile(fs, caPath)
	Expect(err).ToNot(HaveOccurred())
	cd := &cdv2.ComponentDescriptor{}
	Expect(codec.Decode(data, cd)).To(Succeed())
	Expect(os.Setenv(constants.ComponentRepositoryCacheDirEnvVar, "/tmp/components")).To(Succeed())
	repoCtx, err := components.GetOCIRepositoryContext(cd.GetEffectiveRepositoryContext())
	Expect(err).ToNot(HaveOccurred())
	cdCachePath := components.LocalCachePath(repoCtx, cd.Name, cd.Version)
	Expect(fs.MkdirAll(filepath.Dir(cdCachePath), os.ModePerm)).To(Succeed())
	Expect(vfs.WriteFile(fs, cdCachePath, data, os.ModePerm)).To(Succeed())

	Expect(opts.Run(context.TODO(), logr.Discard(), fs)).To(Succeed())

	data, err = vfs.ReadFile(fs, opts.ImageVectorPath)
	Expect(err).ToNot(HaveOccurred())

	imageVector := &iv.ImageVector{}
	Expect(yaml.Unmarshal(data, imageVector)).To(Succeed())
	return imageVector
}
