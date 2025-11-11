// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package pkg_test

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/legacy-image-vector/pkg"
)

var _ = Describe("GenerateOverwrite", func() {

	It("should generate a simple image with tag from a component descriptor", func() {
		imageVector, err := pkg.GenerateImageOverwrite(context.TODO(),
			nil,
			readComponentDescriptor("./testdata/01-component/component-descriptor.yaml"),
			pkg.GenerateImageOverwriteOptions{})
		Expect(err).ToNot(HaveOccurred())

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

	It("should generate a image source with a target version", func() {
		ivReader, err := os.Open("./testdata/resources/10-targetversion.yaml")
		Expect(err).ToNot(HaveOccurred())
		defer func() {
			Expect(ivReader.Close()).ToNot(HaveOccurred())
		}()

		cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
		err = pkg.ParseImageVector(context.TODO(), nil, cd, ivReader, &pkg.ParseImageOptions{})
		Expect(err).ToNot(HaveOccurred())

		imageVector, err := pkg.GenerateImageOverwrite(context.TODO(), nil, cd, pkg.GenerateImageOverwriteOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(imageVector.Images).To(HaveLen(1))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("metrics-server"),
			"Tag":           PointTo(Equal("v0.4.1")),
			"TargetVersion": PointTo(Equal(">= 1.11")),
		})))
	})

	Context("From Component Reference", func() {
		It("should generate image sources from component references", func() {
			ivReader, err := os.Open("./testdata/resources/21-multi-comp-ref.yaml")
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				Expect(ivReader.Close()).ToNot(HaveOccurred())
			}()

			cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
			compRes, err := ctf.NewListResolver(readComponentDescriptors(
				"./testdata/03-autoscaler-0.13.0/component-descriptor.yaml",
				"./testdata/02-autoscaler-0.10.1/component-descriptor.yaml"))
			Expect(err).ToNot(HaveOccurred())
			err = pkg.ParseImageVector(context.TODO(), compRes, cd, ivReader, &pkg.ParseImageOptions{
				ComponentReferencePrefixes: []string{"eu.gcr.io/gardener-project"},
			})
			Expect(err).ToNot(HaveOccurred())

			list := readComponentDescriptors(
				"./testdata/02-autoscaler-0.10.1/component-descriptor.yaml",
				"./testdata/03-autoscaler-0.13.0/component-descriptor.yaml")
			lr, err := ctf.NewListResolver(list)
			Expect(err).ToNot(HaveOccurred())
			imageVector, err := pkg.GenerateImageOverwrite(context.TODO(), lr, cd, pkg.GenerateImageOverwriteOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(imageVector.Images).To(HaveLen(2))
			Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("cluster-autoscaler"),
				"Tag":           PointTo(Equal("sha256:3a33df492c3da1436d7301142d60d1c3e90c354ec70775ac664b8933e4c3d7ec")),
				"TargetVersion": PointTo(Equal(">= 1.16")),
			})))
			Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("cluster-autoscaler"),
				"Tag":           PointTo(Equal("v0.10.1")),
				"TargetVersion": PointTo(Equal("< 1.16")),
			})))
		})

		It("should use the resource name", func() {
			cd := readComponentDescriptor("./testdata/01-component/component-ref-cd.yaml")

			list := readComponentDescriptors(
				"./testdata/02-autoscaler-0.10.1/component-descriptor.yaml",
				"./testdata/03-autoscaler-0.13.0/component-descriptor.yaml")

			// remove labels from the cd resources
			for i := range list.Components {
				for j := range list.Components[i].Resources {
					list.Components[i].Resources[j].SetLabels([]cdv2.Label{})
				}
			}

			lr, err := ctf.NewListResolver(list)
			Expect(err).ToNot(HaveOccurred())
			imageVector, err := pkg.GenerateImageOverwrite(context.TODO(), lr, cd, pkg.GenerateImageOverwriteOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(imageVector.Images).To(HaveLen(2))
			Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("autoscaler"),
				"Tag":           PointTo(Equal("sha256:3a33df492c3da1436d7301142d60d1c3e90c354ec70775ac664b8933e4c3d7ec")),
				"TargetVersion": PointTo(Equal(">= 1.16")),
			})))
			Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("autoscaler"),
				"Tag":           PointTo(Equal("v0.10.1")),
				"TargetVersion": PointTo(Equal("< 1.16")),
			})))
		})

		It("should use a workaround if image and resource names do not match for legacy labels without a resource name", func() {
			cd := readComponentDescriptor("./testdata/01-component/component-ref-cd.yaml")

			list := readComponentDescriptors(
				"./testdata/02-autoscaler-0.10.1/component-descriptor.yaml",
				"./testdata/03-autoscaler-0.13.0/component-descriptor.yaml")
			lr, err := ctf.NewListResolver(list)
			Expect(err).ToNot(HaveOccurred())
			imageVector, err := pkg.GenerateImageOverwrite(context.TODO(), lr, cd, pkg.GenerateImageOverwriteOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(imageVector.Images).To(HaveLen(2))
			Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("autoscaler"),
				"Tag":           PointTo(Equal("sha256:3a33df492c3da1436d7301142d60d1c3e90c354ec70775ac664b8933e4c3d7ec")),
				"TargetVersion": PointTo(Equal(">= 1.16")),
			})))
			Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("autoscaler"),
				"Tag":           PointTo(Equal("v0.10.1")),
				"TargetVersion": PointTo(Equal("< 1.16")),
			})))
		})

		It("should use a workaround use the gardener migration tag if image and resource names do not match for legacy labels without a resource name", func() {
			cd := readComponentDescriptor("./testdata/01-component/legacy-component-ref-cd.yaml")

			list := readComponentDescriptors(
				"./testdata/02-autoscaler-0.10.1/component-descriptor.yaml",
				"./testdata/03-autoscaler-0.13.0/component-descriptor.yaml")
			lr, err := ctf.NewListResolver(list)
			Expect(err).ToNot(HaveOccurred())
			imageVector, err := pkg.GenerateImageOverwrite(context.TODO(), lr, cd, pkg.GenerateImageOverwriteOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(imageVector.Images).To(HaveLen(2))
			Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("autoscaler"),
				"Tag":           PointTo(Equal("sha256:3a33df492c3da1436d7301142d60d1c3e90c354ec70775ac664b8933e4c3d7ec")),
				"TargetVersion": PointTo(Equal(">= 1.16")),
			})))
			Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("autoscaler"),
				"Tag":           PointTo(Equal("v0.10.1")),
				"TargetVersion": PointTo(Equal("< 1.16")),
			})))
		})
	})

	It("should generate image sources from generic images", func() {
		ivReader, err := os.Open("./testdata/resources/30-generic.yaml")
		Expect(err).ToNot(HaveOccurred())
		defer func() {
			Expect(ivReader.Close()).ToNot(HaveOccurred())
		}()

		cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
		err = pkg.ParseImageVector(context.TODO(), nil, cd, ivReader, &pkg.ParseImageOptions{
			GenericDependencies: []string{"hyperkube"},
		})
		Expect(err).ToNot(HaveOccurred())

		list := readComponentDescriptors("./testdata/04-generic-images/component-descriptor.yaml")
		imageVector, err := pkg.GenerateImageOverwrite(context.TODO(), nil, cd, pkg.GenerateImageOverwriteOptions{
			Components: list,
		})
		Expect(err).ToNot(HaveOccurred())

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

	It("should generate image sources from generic images with digest", func() {
		ivReader, err := os.Open("./testdata/resources/30-generic.yaml")
		Expect(err).ToNot(HaveOccurred())
		defer func() {
			Expect(ivReader.Close()).ToNot(HaveOccurred())
		}()

		cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
		err = pkg.ParseImageVector(context.TODO(), nil, cd, ivReader, &pkg.ParseImageOptions{
			GenericDependencies: []string{"hyperkube"},
		})
		Expect(err).ToNot(HaveOccurred())

		list := readComponentDescriptors("./testdata/04-generic-images/component-descriptor-digest.yaml")
		imageVector, err := pkg.GenerateImageOverwrite(context.TODO(), nil, cd, pkg.GenerateImageOverwriteOptions{
			Components: list,
		})
		Expect(err).ToNot(HaveOccurred())

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
			"Tag":           PointTo(Equal("sha256:6db3c05e01e74f85f4cb535181596cea5d0d7cce97cc989e5c11d8ba519b42d3")),
			"TargetVersion": PointTo(Equal("1.18.6")),
		})))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("hyperkube"),
			"Repository":    Equal("k8s.gcr.io/hyperkube"),
			"Tag":           PointTo(Equal("sha256:3a33df492c3da1436d7301142d60d1c3e90c354ec70775ac664b8933e4c3d7ec")),
			"TargetVersion": PointTo(Equal("1.17.10")),
		})))
	})

	It("should generate image sources from generic images based on extra identity", func() {
		ivReader, err := os.Open("./testdata/resources/31-generic.yaml")
		Expect(err).ToNot(HaveOccurred())
		defer func() {
			Expect(ivReader.Close()).ToNot(HaveOccurred())
		}()

		cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
		err = pkg.ParseImageVector(context.TODO(), nil, cd, ivReader, &pkg.ParseImageOptions{
			GenericDependencies: []string{"hyperkube"},
		})
		Expect(err).ToNot(HaveOccurred())

		list := readComponentDescriptors("./testdata/04-generic-images/component-descriptor2.yaml")
		imageVector, err := pkg.GenerateImageOverwrite(context.TODO(), nil, cd, pkg.GenerateImageOverwriteOptions{
			Components: list,
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(imageVector.Images).To(HaveLen(2))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("kube-apiserver"),
			"Repository":    Equal("eu.gcr.io/gardener-project/hyperkube"),
			"Tag":           PointTo(Equal("v1.16.15-mod1")),
			"TargetVersion": PointTo(Equal("1.16.15")),
		})))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("kube-apiserver"),
			"Repository":    Equal("eu.gcr.io/gardener-project/hyperkube"),
			"Tag":           PointTo(Equal("v1.15.12-mod1")),
			"TargetVersion": PointTo(Equal("1.15.12")),
		})))
	})

	It("should generate a simple image with digest from a component descriptor", func() {
		imageVector, err := pkg.GenerateImageOverwrite(context.TODO(),
			nil,
			readComponentDescriptor("./testdata/01-component/component-descriptor.yaml"),
			pkg.GenerateImageOverwriteOptions{
				ReplaceWithDigests: true,
				OciClient: fakeResolver{
					resolve: func(ctx context.Context, ref string) (name string, desc ocispecv1.Descriptor, err error) {
						dig := digest.NewDigestFromBytes(digest.SHA256, []byte("abc"))
						return "", ocispecv1.Descriptor{Digest: dig}, nil
					},
				},
			})
		Expect(err).ToNot(HaveOccurred())

		Expect(imageVector.Images).To(HaveLen(3))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name": Equal("pause-container"),
			"Tag":  PointTo(Equal("sha256:616263")),
		})))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name": Equal("pause-container"),
			"Tag":  PointTo(Equal("sha256:eb9086d472747453ad2d5cfa10f80986d9b0afb9ae9c4256fe2887b029566d06")),
		})))
		Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Name": Equal("gardenlet"),
			"Tag":  PointTo(Equal("sha256:616263")),
		})))
	})

})

type fakeResolver struct {
	resolve func(ctx context.Context, ref string) (name string, desc ocispecv1.Descriptor, err error)
}

func (r fakeResolver) Resolve(ctx context.Context, ref string) (name string, desc ocispecv1.Descriptor, err error) {
	return r.resolve(ctx, ref)
}
