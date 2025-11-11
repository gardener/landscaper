// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package pkg_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/legacy-image-vector/pkg"
)

var _ = Describe("Add", func() {

	It("should add a image source with tag", func() {

		ivReader, err := os.Open("./testdata/resources/00-tag.yaml")
		Expect(err).ToNot(HaveOccurred())
		defer func() {
			Expect(ivReader.Close()).ToNot(HaveOccurred())
		}()

		cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
		err = pkg.ParseImageVector(context.TODO(), nil, cd, ivReader, &pkg.ParseImageOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ExternalRelation),
		}))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("pause-container"),
			"Version":       Equal("3.1"),
			"ExtraIdentity": HaveKeyWithValue(pkg.TagExtraIdentity, "3.1"),
			"Labels": ContainElements(
				cdv2.Label{
					Name:  pkg.NameLabel,
					Value: json.RawMessage(`"pause-container"`),
				},
				cdv2.Label{
					Name:  pkg.RepositoryLabel,
					Value: json.RawMessage(`"gcr.io/google_containers/pause-amd64"`),
				},
				cdv2.Label{
					Name:  pkg.SourceRepositoryLabel,
					Value: json.RawMessage(`"github.com/kubernetes/kubernetes/blob/master/build/pause/Dockerfile"`),
				},
			),
		}))
		Expect(cd.Resources[0].Access.Object).To(MatchKeys(IgnoreExtras, Keys{
			"imageReference": Equal("gcr.io/google_containers/pause-amd64:3.1"),
		}))
	})

	It("should add a image source with a digest as tag", func() {

		ivReader, err := os.Open("./testdata/resources/03-sha.yaml")
		Expect(err).ToNot(HaveOccurred())
		defer func() {
			Expect(ivReader.Close()).ToNot(HaveOccurred())
		}()

		cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
		err = pkg.ParseImageVector(context.TODO(), nil, cd, ivReader, &pkg.ParseImageOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ExternalRelation),
		}))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("pause-container"),
			"Version":       Equal("v0.0.0"),
			"ExtraIdentity": HaveKeyWithValue(pkg.TagExtraIdentity, "sha256:179e67c248007299e05791db36298c41cbf0992372204a68473e12795a51b06b"),
			"Labels": ContainElements(
				cdv2.Label{
					Name:  pkg.NameLabel,
					Value: json.RawMessage(`"pause-container"`),
				},
				cdv2.Label{
					Name:  pkg.RepositoryLabel,
					Value: json.RawMessage(`"gcr.io/google_containers/pause-amd64"`),
				},
				cdv2.Label{
					Name:  pkg.SourceRepositoryLabel,
					Value: json.RawMessage(`"github.com/kubernetes/kubernetes/blob/master/build/pause/Dockerfile"`),
				},
			),
		}))
		Expect(cd.Resources[0].Access.Object).To(MatchKeys(IgnoreExtras, Keys{
			"imageReference": Equal("gcr.io/google_containers/pause-amd64@sha256:179e67c248007299e05791db36298c41cbf0992372204a68473e12795a51b06b"),
		}))
	})

	It("should add a image source with a label", func() {

		ivReader, err := os.Open("./testdata/resources/01-labels.yaml")
		Expect(err).ToNot(HaveOccurred())
		defer func() {
			Expect(ivReader.Close()).ToNot(HaveOccurred())
		}()

		cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
		err = pkg.ParseImageVector(context.TODO(), nil, cd, ivReader, &pkg.ParseImageOptions{})
		Expect(err).ToNot(HaveOccurred())

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
					Name:  pkg.NameLabel,
					Value: json.RawMessage(`"pause-container"`),
				},
				cdv2.Label{
					Name:  pkg.RepositoryLabel,
					Value: json.RawMessage(`"gcr.io/google_containers/pause-amd64"`),
				},
				cdv2.Label{
					Name:  pkg.SourceRepositoryLabel,
					Value: json.RawMessage(`"github.com/kubernetes/kubernetes/blob/master/build/pause/Dockerfile"`),
				},
			),
		}))
	})

	It("should add imagevector labels for inline image definitions", func() {

		ivReader, err := os.Open("./testdata/resources/02-inline.yaml")
		Expect(err).ToNot(HaveOccurred())
		defer func() {
			Expect(ivReader.Close()).ToNot(HaveOccurred())
		}()

		cd := readComponentDescriptor("./testdata/05-inline/component-descriptor.yaml")
		err = pkg.ParseImageVector(context.TODO(), nil, cd, ivReader, &pkg.ParseImageOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.LocalRelation),
		}))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("gardenlet"),
			"Version": Equal("v0.0.0"),
			"Labels": ContainElements(
				cdv2.Label{
					Name:  pkg.NameLabel,
					Value: json.RawMessage(`"gardenlet"`),
				},
				cdv2.Label{
					Name:  pkg.RepositoryLabel,
					Value: json.RawMessage(`"eu.gcr.io/gardener-project/gardener/gardenlet"`),
				},
				cdv2.Label{
					Name:  pkg.SourceRepositoryLabel,
					Value: json.RawMessage(`"github.com/gardener/gardener"`),
				},
			),
		}))
	})

	It("should add a image source with tag and target version", func() {

		ivReader, err := os.Open("./testdata/resources/10-targetversion.yaml")
		Expect(err).ToNot(HaveOccurred())
		defer func() {
			Expect(ivReader.Close()).ToNot(HaveOccurred())
		}()

		cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
		err = pkg.ParseImageVector(context.TODO(), nil, cd, ivReader, &pkg.ParseImageOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(cd.Resources).To(HaveLen(1))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ExternalRelation),
		}))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("metrics-server"),
			"Version":       Equal("v0.4.1"),
			"ExtraIdentity": HaveKeyWithValue(pkg.TagExtraIdentity, "v0.4.1"),
		}))
	})

	It("should add two image sources with different target versions", func() {

		ivReader, err := os.Open("./testdata/resources/11-multi-targetversion.yaml")
		Expect(err).ToNot(HaveOccurred())
		defer func() {
			Expect(ivReader.Close()).ToNot(HaveOccurred())
		}()

		cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
		err = pkg.ParseImageVector(context.TODO(), nil, cd, ivReader, &pkg.ParseImageOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(cd.Resources).To(HaveLen(2))
		Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ExternalRelation),
		}))
		Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("metrics-server"),
			"Version":       Equal("v0.4.1"),
			"ExtraIdentity": HaveKeyWithValue(pkg.TagExtraIdentity, "v0.4.1"),
		}))

		Expect(cd.Resources[1]).To(MatchFields(IgnoreExtras, Fields{
			"Relation": Equal(cdv2.ExternalRelation),
		}))
		Expect(cd.Resources[1].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":          Equal("metrics-server"),
			"Version":       Equal("v0.3.1"),
			"ExtraIdentity": HaveKeyWithValue(pkg.TagExtraIdentity, "v0.3.1"),
		}))
	})

	It("should throw an error in case of different target version labels for the same name and tag", func() {
		ivReader, err := os.Open("./testdata/resources/12-multi-targetversion-same-name-and-tag.yaml")
		Expect(err).ToNot(HaveOccurred())
		defer func() {
			Expect(ivReader.Close()).ToNot(HaveOccurred())
		}()

		cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")

		err = pkg.ParseImageVector(context.TODO(), nil, cd, ivReader, &pkg.ParseImageOptions{})
		Expect(err).To(HaveOccurred())
	})

	Context("ComponentReferences", func() {

		Context("should add image sources that match a given pattern as component reference", func() {

			It("match by resource name", func() {
				ivReader, err := os.Open("./testdata/resources/20-0-comp-ref.yaml")
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					Expect(ivReader.Close()).ToNot(HaveOccurred())
				}()

				cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
				compRes, err := ctf.NewListResolver(readComponentDescriptors("./testdata/02-autoscaler-0.10.1/component-descriptor.yaml"))
				Expect(err).ToNot(HaveOccurred())
				err = pkg.ParseImageVector(context.TODO(),
					compRes,
					cd,
					ivReader,
					&pkg.ParseImageOptions{
						ComponentReferencePrefixes: []string{"eu.gcr.io/gardener-project/gardener"},
					},
				)
				Expect(err).ToNot(HaveOccurred())

				Expect(cd.Resources).To(HaveLen(0))
				Expect(cd.ComponentReferences).To(HaveLen(1))
				Expect(cd.ComponentReferences[0]).To(MatchFields(IgnoreExtras, Fields{
					"Name":          Equal("cluster-autoscaler"),
					"ComponentName": Equal("github.com/gardener/autoscaler"),
					"Version":       Equal("v0.10.1"),
					"ExtraIdentity": HaveKeyWithValue("imagevector-gardener-cloud+tag", "v0.10.1"),
				}))

				imageLabelBytes, ok := cd.ComponentReferences[0].GetLabels().Get(pkg.ImagesLabel)
				Expect(ok).To(BeTrue())
				imageVector := &pkg.ComponentReferenceImageVector{}
				Expect(json.Unmarshal(imageLabelBytes, imageVector)).To(Succeed())
				Expect(imageVector.Images).To(HaveLen(1))
				Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
					"ImageEntry": MatchFields(IgnoreExtras, Fields{
						"Name": Equal("cluster-autoscaler"),
						"Tag":  PointTo(Equal("v0.10.1")),
					}),
					"ResourceID": MatchKeys(0, Keys{
						"name": Equal("cluster-autoscaler"),
					}),
				})))
			})

			It("match by image repository", func() {
				ivReader, err := os.Open("./testdata/resources/20-1-comp-ref.yaml")
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					Expect(ivReader.Close()).ToNot(HaveOccurred())
				}()

				cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
				compRes, err := ctf.NewListResolver(readComponentDescriptors("./testdata/02-autoscaler-0.10.1/component-descriptor.yaml"))
				Expect(err).ToNot(HaveOccurred())
				err = pkg.ParseImageVector(context.TODO(),
					compRes,
					cd,
					ivReader,
					&pkg.ParseImageOptions{
						ComponentReferencePrefixes: []string{"eu.gcr.io/gardener-project/gardener"},
					},
				)
				Expect(err).ToNot(HaveOccurred())

				Expect(cd.Resources).To(HaveLen(0))
				Expect(cd.ComponentReferences).To(HaveLen(1))
				Expect(cd.ComponentReferences[0]).To(MatchFields(IgnoreExtras, Fields{
					"Name":          Equal("autoscaler"),
					"ComponentName": Equal("github.com/gardener/autoscaler"),
					"Version":       Equal("v0.10.1"),
					"ExtraIdentity": HaveKeyWithValue("imagevector-gardener-cloud+tag", "v0.10.1"),
				}))

				imageLabelBytes, ok := cd.ComponentReferences[0].GetLabels().Get(pkg.ImagesLabel)
				Expect(ok).To(BeTrue())
				imageVector := &pkg.ComponentReferenceImageVector{}
				Expect(json.Unmarshal(imageLabelBytes, imageVector)).To(Succeed())
				Expect(imageVector.Images).To(HaveLen(1))
				Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
					"ImageEntry": MatchFields(IgnoreExtras, Fields{
						"Name": Equal("autoscaler"),
						"Tag":  PointTo(Equal("v0.10.1")),
					}),
					"ResourceID": MatchKeys(0, Keys{
						"name": Equal("cluster-autoscaler"),
					}),
				})))
			})

		})

		It("should return an error if the referenced component cannot be found in the context", func() {
			ivReader, err := os.Open("./testdata/resources/20-0-comp-ref.yaml")
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				Expect(ivReader.Close()).ToNot(HaveOccurred())
			}()

			cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
			compRes, err := ctf.NewListResolver(readComponentDescriptors())
			Expect(err).ToNot(HaveOccurred())
			err = pkg.ParseImageVector(context.TODO(),
				compRes,
				cd,
				ivReader,
				&pkg.ParseImageOptions{
					ComponentReferencePrefixes: []string{"eu.gcr.io/gardener-project/gardener"},
				},
			)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ctf.NotFoundError)).To(BeTrue())
		})

		It("should add image sources with the component reference label as component reference", func() {

			ivReader, err := os.Open("./testdata/resources/23-comp-ref-label.yaml")
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				Expect(ivReader.Close()).ToNot(HaveOccurred())
			}()

			cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
			compRes, err := ctf.NewListResolver(readComponentDescriptors("./testdata/02-autoscaler-0.10.1/component-descriptor.yaml"))
			Expect(err).ToNot(HaveOccurred())
			err = pkg.ParseImageVector(context.TODO(), compRes, cd, ivReader, &pkg.ParseImageOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(cd.Resources).To(HaveLen(0))
			Expect(cd.ComponentReferences).To(HaveLen(1))
			Expect(cd.ComponentReferences[0]).To(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("cluster-autoscaler"),
				"ComponentName": Equal("github.com/gardener/autoscaler"),
				"Version":       Equal("v0.10.1"),
				"ExtraIdentity": HaveKeyWithValue("imagevector-gardener-cloud+tag", "v0.10.1"),
			}))

			imageLabelBytes, ok := cd.ComponentReferences[0].GetLabels().Get(pkg.ImagesLabel)
			Expect(ok).To(BeTrue())
			imageVector := &pkg.ImageVector{}
			Expect(json.Unmarshal(imageLabelBytes, imageVector)).To(Succeed())
			Expect(imageVector.Images).To(HaveLen(1))
			Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name": Equal("cluster-autoscaler"),
				"Tag":  PointTo(Equal("v0.10.1")),
			})))
		})

		It("should add image sources with the component reference label and overwrites as component reference", func() {

			ivReader, err := os.Open("./testdata/resources/24-comp-ref-label.yaml")
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				Expect(ivReader.Close()).ToNot(HaveOccurred())
			}()

			cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
			compRes, err := ctf.NewListResolver(readComponentDescriptors("./testdata/03-autoscaler-0.13.0/component-descriptor.yaml"))
			Expect(err).ToNot(HaveOccurred())
			err = pkg.ParseImageVector(context.TODO(), compRes, cd, ivReader, &pkg.ParseImageOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(cd.Resources).To(HaveLen(0))
			Expect(cd.ComponentReferences).To(HaveLen(1))
			Expect(cd.ComponentReferences[0]).To(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("cla"),
				"ComponentName": Equal("github.com/gardener/autoscaler"),
				"Version":       Equal("v0.13.0"),
				"ExtraIdentity": HaveKeyWithValue("imagevector-gardener-cloud+tag", "v0.13.0"),
			}))

			imageLabelBytes, ok := cd.ComponentReferences[0].GetLabels().Get(pkg.ImagesLabel)
			Expect(ok).To(BeTrue())
			imageVector := &pkg.ImageVector{}
			Expect(json.Unmarshal(imageLabelBytes, imageVector)).To(Succeed())
			Expect(imageVector.Images).To(HaveLen(1))
			Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name": Equal("cluster-autoscaler"),
				"Tag":  PointTo(Equal("v0.10.1")),
			})))
		})

		It("should not add image sources that match a given pattern as component reference but has a ignore label", func() {

			ivReader, err := os.Open("./testdata/resources/25-comp-ref-ignore-label.yaml")
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				Expect(ivReader.Close()).ToNot(HaveOccurred())
			}()

			cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
			compRes, err := ctf.NewListResolver(readComponentDescriptors())
			Expect(err).ToNot(HaveOccurred())
			err = pkg.ParseImageVector(context.TODO(), compRes, cd, ivReader, &pkg.ParseImageOptions{
				ComponentReferencePrefixes: []string{"eu.gcr.io/gardener-project/gardener"},
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(cd.Resources).To(HaveLen(1))
			Expect(cd.ComponentReferences).To(HaveLen(0))
		})

		It("should not add a image sources that match a given pattern as component reference but is excluded", func() {

			ivReader, err := os.Open("./testdata/resources/20-0-comp-ref.yaml")
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				Expect(ivReader.Close()).ToNot(HaveOccurred())
			}()

			cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
			compRes, err := ctf.NewListResolver(readComponentDescriptors())
			Expect(err).ToNot(HaveOccurred())
			err = pkg.ParseImageVector(context.TODO(), compRes, cd, ivReader, &pkg.ParseImageOptions{
				ComponentReferencePrefixes: []string{"eu.gcr.io/gardener-project/gardener"},
				ExcludeComponentReference:  []string{"cluster-autoscaler"},
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(cd.ComponentReferences).To(HaveLen(0))
			Expect(cd.Resources).To(HaveLen(1))
			Expect(cd.Resources[0]).To(MatchFields(IgnoreExtras, Fields{
				"Relation": Equal(cdv2.ExternalRelation),
			}))
			Expect(cd.Resources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("cluster-autoscaler"),
				"Version":       Equal("v0.10.1"),
				"ExtraIdentity": HaveKeyWithValue(pkg.TagExtraIdentity, "v0.10.1"),
			}))
		})

		It("should add two image sources that match a given pattern as one component reference", func() {

			ivReader, err := os.Open("./testdata/resources/22-multi-image-comp-ref.yaml")
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				Expect(ivReader.Close()).ToNot(HaveOccurred())
			}()

			cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
			compRes, err := ctf.NewListResolver(readComponentDescriptors("./testdata/03-autoscaler-0.13.0/component-descriptor.yaml"))
			Expect(err).ToNot(HaveOccurred())
			err = pkg.ParseImageVector(context.TODO(), compRes, cd, ivReader, &pkg.ParseImageOptions{
				ComponentReferencePrefixes: []string{"eu.gcr.io/gardener-project/gardener"},
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(cd.Resources).To(HaveLen(0))
			Expect(cd.ComponentReferences).To(HaveLen(1))
			Expect(cd.ComponentReferences[0]).To(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("cluster-autoscaler"),
				"ComponentName": Equal("github.com/gardener/autoscaler"),
				"Version":       Equal("v0.13.0"),
				"ExtraIdentity": And(HaveKey(pkg.TagExtraIdentity), Not(HaveKey("name"))),
			}))

			imageLabelBytes, ok := cd.ComponentReferences[0].GetLabels().Get(pkg.ImagesLabel)
			Expect(ok).To(BeTrue())
			imageVector := &pkg.ImageVector{}
			Expect(json.Unmarshal(imageLabelBytes, imageVector)).To(Succeed())
			Expect(imageVector.Images).To(HaveLen(2))
			Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name":       Equal("cluster-autoscaler"),
				"Repository": Equal("eu.gcr.io/gardener-project/gardener/autoscaler/cluster-autoscaler"),
				"Tag":        PointTo(Equal("v0.13.0")),
			})))
			Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name":       Equal("cluster-autoscaler"),
				"Repository": Equal("eu.gcr.io/gardener-project/gardener/autoscaler/old"),
				"Tag":        PointTo(Equal("v0.13.0")),
			})))
		})
	})

	Context("Generic Dependencies", func() {

		It("should add generic sources that match a given generic dependency name", func() {
			ivReader, err := os.Open("./testdata/resources/30-generic.yaml")
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				Expect(ivReader.Close()).ToNot(HaveOccurred())
			}()

			cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
			err = pkg.ParseImageVector(context.TODO(), nil, cd, ivReader, &pkg.ParseImageOptions{
				GenericDependencies: []string{
					"hyperkube",
				},
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(cd.Resources).To(HaveLen(0))
			Expect(cd.ComponentReferences).To(HaveLen(0))

			imageLabelBytes, ok := cd.GetLabels().Get(pkg.ImagesLabel)
			Expect(ok).To(BeTrue())
			imageVector := &pkg.ImageVector{}
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

		It("should add an image entry as generic resources hwn a tag is absent", func() {
			ivReader, err := os.Open("./testdata/resources/30-generic.yaml")
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				Expect(ivReader.Close()).ToNot(HaveOccurred())
			}()

			cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
			err = pkg.ParseImageVector(context.TODO(), nil, cd, ivReader, &pkg.ParseImageOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(cd.Resources).To(HaveLen(0))
			Expect(cd.ComponentReferences).To(HaveLen(0))

			imageLabelBytes, ok := cd.GetLabels().Get(pkg.ImagesLabel)
			Expect(ok).To(BeTrue())
			imageVector := &pkg.ImageVector{}
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

		It("should add generic sources that match a given generic dependency name defined by a list of dependencies", func() {

			ivReader, err := os.Open("./testdata/resources/30-generic.yaml")
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				Expect(ivReader.Close()).ToNot(HaveOccurred())
			}()

			cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
			err = pkg.ParseImageVector(context.TODO(), nil, cd, ivReader, &pkg.ParseImageOptions{
				GenericDependencies: []string{
					"hyperkube",
				},
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(cd.Resources).To(HaveLen(0))
			Expect(cd.ComponentReferences).To(HaveLen(0))

			imageLabelBytes, ok := cd.GetLabels().Get(pkg.ImagesLabel)
			Expect(ok).To(BeTrue())
			imageVector := &pkg.ImageVector{}
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

		It("should add generic sources that are labeled", func() {
			ivReader, err := os.Open("./testdata/resources/31-generic-labels.yaml")
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				Expect(ivReader.Close()).ToNot(HaveOccurred())
			}()

			cd := readComponentDescriptor("./testdata/00-component/component-descriptor.yaml")
			err = pkg.ParseImageVector(context.TODO(), nil, cd, ivReader, &pkg.ParseImageOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(cd.Resources).To(HaveLen(1))
			Expect(cd.ComponentReferences).To(HaveLen(0))

			imageLabelBytes, ok := cd.GetLabels().Get(pkg.ImagesLabel)
			Expect(ok).To(BeTrue())
			imageVector := &pkg.ImageVector{}
			Expect(json.Unmarshal(imageLabelBytes, imageVector)).To(Succeed())
			Expect(imageVector.Images).To(HaveLen(1))
			Expect(imageVector.Images).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name":          Equal("hyperkube"),
				"Repository":    Equal("eu.gcr.io/gardener-project/new/hyperkube"),
				"TargetVersion": PointTo(Equal("< 1.19")),
			})))
		})

	})

})
