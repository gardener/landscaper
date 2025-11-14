// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imagevector_test

import (
	"context"
	"encoding/json"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	iv "github.com/gardener/landscaper/legacy-image-vector/pkg"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/codec"

	ivcmd "github.com/gardener/landscaper/legacy-component-cli/pkg/commands/imagevector"
)

var _ = ginkgo.Describe("Add", func() {

	var testdataFs vfs.FileSystem

	ginkgo.BeforeEach(func() {
		fs, err := projectionfs.New(osfs.New(), "./testdata")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), fs)
	})

	ginkgo.It("should add a image source with tag", func() {

		opts := &ivcmd.AddOptions{
			ComponentDescriptorPath: "./00-component/component-descriptor.yaml",
			ImageVectorPath:         "./resources/00-tag.yaml",
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, opts.ComponentDescriptorPath)
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ExternalRelation),
		}))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("pause-container"),
			"Version":       Equal("3.1"),
			"ExtraIdentity": HaveKeyWithValue(iv.TagExtraIdentity, "3.1"),
			"Labels": ContainElements(
				cdv2.Label{
					Name:  iv.NameLabel,
					Value: json.RawMessage(`"pause-container"`),
				},
				cdv2.Label{
					Name:  iv.RepositoryLabel,
					Value: json.RawMessage(`"gcr.io/google_containers/pause-amd64"`),
				},
				cdv2.Label{
					Name:  iv.SourceRepositoryLabel,
					Value: json.RawMessage(`"github.com/kubernetes/kubernetes/blob/master/build/pause/Dockerfile"`),
				},
			),
		}))
		Expect(cd.Resources[0].Access.Object).To(MatchKeys(IgnoreExtras, Keys{
			"imageReference": Equal("gcr.io/google_containers/pause-amd64:3.1"),
		}))
	})

	ginkgo.It("should add a image source with a digest as tag", func() {

		opts := &ivcmd.AddOptions{
			ComponentDescriptorPath: "./00-component/component-descriptor.yaml",
			ImageVectorPath:         "./resources/03-sha.yaml",
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, opts.ComponentDescriptorPath)
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ExternalRelation),
		}))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("pause-container"),
			"Version":       Equal("v0.0.0"),
			"ExtraIdentity": HaveKeyWithValue(iv.TagExtraIdentity, "sha256:179e67c248007299e05791db36298c41cbf0992372204a68473e12795a51b06b"),
			"Labels": ContainElements(
				cdv2.Label{
					Name:  iv.NameLabel,
					Value: json.RawMessage(`"pause-container"`),
				},
				cdv2.Label{
					Name:  iv.RepositoryLabel,
					Value: json.RawMessage(`"gcr.io/google_containers/pause-amd64"`),
				},
				cdv2.Label{
					Name:  iv.SourceRepositoryLabel,
					Value: json.RawMessage(`"github.com/kubernetes/kubernetes/blob/master/build/pause/Dockerfile"`),
				},
			),
		}))
		Expect(cd.Resources[0].Access.Object).To(MatchKeys(IgnoreExtras, Keys{
			"imageReference": Equal("gcr.io/google_containers/pause-amd64@sha256:179e67c248007299e05791db36298c41cbf0992372204a68473e12795a51b06b"),
		}))
	})

	ginkgo.It("should add a image source with a label", func() {

		opts := &ivcmd.AddOptions{
			ComponentDescriptorPath: "./00-component/component-descriptor.yaml",
			ImageVectorPath:         "./resources/01-labels.yaml",
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, opts.ComponentDescriptorPath)
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ExternalRelation),
		}))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("pause-container"),
			"Version": Equal("3.1"),
			"Labels": ContainElements(
				cdv2.Label{
					Name:  "my-label",
					Value: json.RawMessage(`"myval"`),
				},
				cdv2.Label{
					Name:  iv.NameLabel,
					Value: json.RawMessage(`"pause-container"`),
				},
				cdv2.Label{
					Name:  iv.RepositoryLabel,
					Value: json.RawMessage(`"gcr.io/google_containers/pause-amd64"`),
				},
				cdv2.Label{
					Name:  iv.SourceRepositoryLabel,
					Value: json.RawMessage(`"github.com/kubernetes/kubernetes/blob/master/build/pause/Dockerfile"`),
				},
			),
		}))
	})

	ginkgo.It("should add imagevector labels for inline image definitions", func() {

		opts := &ivcmd.AddOptions{
			ComponentDescriptorPath: "./05-inline/component-descriptor.yaml",
			ImageVectorPath:         "./resources/02-inline.yaml",
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, opts.ComponentDescriptorPath)
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.LocalRelation),
		}))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("gardenlet"),
			"Version": Equal("v0.0.0"),
			"Labels": ContainElements(
				cdv2.Label{
					Name:  iv.NameLabel,
					Value: json.RawMessage(`"gardenlet"`),
				},
				cdv2.Label{
					Name:  iv.RepositoryLabel,
					Value: json.RawMessage(`"eu.gcr.io/gardener-project/gardener/gardenlet"`),
				},
				cdv2.Label{
					Name:  iv.SourceRepositoryLabel,
					Value: json.RawMessage(`"github.com/gardener/gardener"`),
				},
			),
		}))
	})

	ginkgo.It("should add a image source with tag and target version", func() {

		opts := &ivcmd.AddOptions{
			ComponentDescriptorPath: "./00-component/component-descriptor.yaml",
			ImageVectorPath:         "./resources/10-targetversion.yaml",
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, opts.ComponentDescriptorPath)
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ExternalRelation),
		}))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("metrics-server"),
			"Version":       Equal("v0.4.1"),
			"ExtraIdentity": HaveKeyWithValue(iv.TagExtraIdentity, "v0.4.1"),
		}))
	})

	ginkgo.It("should add two image sources with different target versions", func() {

		opts := &ivcmd.AddOptions{
			ComponentDescriptorPath: "./00-component/component-descriptor.yaml",
			ImageVectorPath:         "./resources/11-multi-targetversion.yaml",
		}

		Expect(opts.Run(context.TODO(), logr.Discard(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, opts.ComponentDescriptorPath)
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Resources).To(HaveLen(2))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ExternalRelation),
		}))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("metrics-server"),
			"Version":       Equal("v0.4.1"),
			"ExtraIdentity": HaveKeyWithValue(iv.TagExtraIdentity, "v0.4.1"),
		}))

		Expect(cd.Resources[1]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ExternalRelation),
		}))
		Expect(cd.Resources[1].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("metrics-server"),
			"Version":       Equal("v0.3.1"),
			"ExtraIdentity": HaveKeyWithValue(iv.TagExtraIdentity, "v0.3.1"),
		}))
	})

	ginkgo.Context("Generic Dependencies", func() {

		ginkgo.It("should add generic sources that match a given generic dependency name", func() {
			opts := &ivcmd.AddOptions{
				ParseImageOptions: iv.ParseImageOptions{
					GenericDependencies: []string{
						"hyperkube",
					},
				},
			}
			cd := runAdd(testdataFs, "./00-component/component-descriptor.yaml", "./resources/30-generic.yaml", opts)

			Expect(cd.Resources).To(HaveLen(0))
			Expect(cd.ComponentReferences).To(HaveLen(0))

			imageLabelBytes, ok := cd.GetLabels().Get(iv.ImagesLabel)
			Expect(ok).To(BeTrue())
			imageVector := &iv.ImageVector{}
			Expect(json.Unmarshal(imageLabelBytes, imageVector)).To(Succeed())
			Expect(imageVector.Images).To(HaveLen(2))
			Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("hyperkube"),
				"Repository":    Equal("k8s.gcr.io/hyperkube"),
				"TargetVersion": PointTo(Equal("< 1.19")),
			})))
			Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("hyperkube"),
				"Repository":    Equal("eu.gcr.io/gardener-project/hyperkube"),
				"TargetVersion": PointTo(Equal(">= 1.19")),
			})))
		})

		ginkgo.It("should add generic sources that match a given generic dependency name defined by a list of dependencies", func() {
			opts := &ivcmd.AddOptions{
				GenericDependencies: "hyperkube",
			}
			cd := runAdd(testdataFs, "./00-component/component-descriptor.yaml", "./resources/30-generic.yaml", opts)

			Expect(cd.Resources).To(HaveLen(0))
			Expect(cd.ComponentReferences).To(HaveLen(0))

			imageLabelBytes, ok := cd.GetLabels().Get(iv.ImagesLabel)
			Expect(ok).To(BeTrue())
			imageVector := &iv.ImageVector{}
			Expect(json.Unmarshal(imageLabelBytes, imageVector)).To(Succeed())
			Expect(imageVector.Images).To(HaveLen(2))
			Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("hyperkube"),
				"Repository":    Equal("k8s.gcr.io/hyperkube"),
				"TargetVersion": PointTo(Equal("< 1.19")),
			})))
			Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("hyperkube"),
				"Repository":    Equal("eu.gcr.io/gardener-project/hyperkube"),
				"TargetVersion": PointTo(Equal(">= 1.19")),
			})))
		})
	})

})

// runAdd runs the add command
func runAdd(fs vfs.FileSystem, caPath, ivPath string, addOpts ...*ivcmd.AddOptions) *cdv2.ComponentDescriptor {
	Expect(len(addOpts) <= 1).To(BeTrue())
	opts := &ivcmd.AddOptions{}
	if len(addOpts) == 1 {
		opts = addOpts[0]
	}
	opts.ComponentDescriptorPath = caPath
	opts.ImageVectorPath = ivPath
	Expect(opts.Complete(nil)).To(Succeed())

	Expect(opts.Run(context.TODO(), logr.Discard(), fs)).To(Succeed())

	data, err := vfs.ReadFile(fs, opts.ComponentDescriptorPath)
	Expect(err).ToNot(HaveOccurred())

	cd := &cdv2.ComponentDescriptor{}
	Expect(codec.Decode(data, cd)).To(Succeed())
	return cd
}
