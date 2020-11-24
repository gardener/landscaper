// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cdutils_test

import (
	"testing"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"

	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "landscaper component descriptor")
}

var _ = Describe("mapped component descriptor", func() {

	Context("#ResolvedComponentDescriptor", func() {

		resolveFunc := func(meta cdv2.ComponentReference) (cdv2.ComponentDescriptor, error) {
			return cdv2.ComponentDescriptor{
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name:    meta.ComponentName,
						Version: meta.Version,
					},
				},
			}, nil
		}

		testResources := []cdv2.Resource{
			{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "r1",
					Version: "1.5.5",
				},
			},
			{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "r2",
					Version: "1.5.0",
				},
			},
		}

		testSources := []cdv2.Source{
			{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name: "s1",
				},
				Access: cdv2.NewUnstructuredType("custom", nil),
			},
			{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name: "s2",
				},
				Access: cdv2.NewUnstructuredType("custom", nil),
			},
		}

		It("should map default metadata", func() {
			cd := cdv2.ComponentDescriptor{}
			cd.ObjectMeta = cdv2.ObjectMeta{
				Name:    "comp",
				Version: "1.0.0",
			}
			cd.Provider = cdv2.ExternalProvider
			cd.RepositoryContexts = []cdv2.RepositoryContext{
				{
					BaseURL: "http://example.com",
				},
			}

			mcd, err := cdutils.ConvertFromComponentDescriptor(cd, resolveFunc)
			Expect(err).ToNot(HaveOccurred())
			Expect(mcd.Name).To(Equal("comp"))
			Expect(mcd.Version).To(Equal("1.0.0"))
			Expect(mcd.Provider).To(Equal(cdv2.ExternalProvider))
			Expect(mcd.RepositoryContexts).To(ConsistOf(cd.RepositoryContexts[0]))
		})

		It("should convert a list sources to a map by the resource name", func() {
			cd := cdv2.ComponentDescriptor{}
			cd.Sources = testSources

			mcd, err := cdutils.ConvertFromComponentDescriptor(cd, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(mcd.Sources).To(HaveKeyWithValue("s1", testSources[0]))
			Expect(mcd.Sources).To(HaveKeyWithValue("s2", testSources[1]))
		})

		It("should convert a list resources to a map by the resource name", func() {
			cd := cdv2.ComponentDescriptor{}
			cd.Resources = testResources

			mcd, err := cdutils.ConvertFromComponentDescriptor(cd, resolveFunc)
			Expect(err).ToNot(HaveOccurred())
			Expect(mcd.Resources).To(HaveKeyWithValue("r1", testResources[0]))
			Expect(mcd.Resources).To(HaveKeyWithValue("r2", testResources[1]))
		})

		It("should convert a list component references to a map by the reference's name", func() {
			cd := cdv2.ComponentDescriptor{}
			cd.ComponentReferences = []cdv2.ComponentReference{
				{
					Name:          "ref1",
					ComponentName: "comp1",
					Version:       "1.0.0",
				},
				{
					Name:          "ref2",
					ComponentName: "comp2",
					Version:       "1.0.1",
				},
			}

			mcd, err := cdutils.ConvertFromComponentDescriptor(cd, resolveFunc)
			Expect(err).ToNot(HaveOccurred())
			Expect(mcd.ComponentReferences).To(HaveKeyWithValue("ref1", gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"ResolvedComponentSpec": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"ObjectMeta": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Name":    Equal("comp1"),
						"Version": Equal("1.0.0"),
					}),
				}),
			})))
			Expect(mcd.ComponentReferences).To(HaveKeyWithValue("ref2", gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"ResolvedComponentSpec": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"ObjectMeta": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Name":    Equal("comp2"),
						"Version": Equal("1.0.1"),
					}),
				}),
			})))
		})
	})

})
