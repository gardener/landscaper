// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package filters_test

import (
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"

	filter "github.com/gardener/landscaper/legacy-component-cli/pkg/transport/filters"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Filters Test Suite")
}

var _ = ginkgo.Describe("filters", func() {

	ginkgo.Context("accessTypeFilter", func() {

		ginkgo.It("should match if access type is in include list", func() {
			cd := cdv2.ComponentDescriptor{}
			res := cdv2.Resource{
				Access: cdv2.NewEmptyUnstructured(cdv2.OCIRegistryType),
			}
			spec := filter.AccessTypeFilterSpec{
				IncludeAccessTypes: []string{
					cdv2.OCIRegistryType,
				},
			}

			f, err := filter.NewAccessTypeFilter(spec)
			Expect(err).ToNot(HaveOccurred())

			actualMatch := f.Matches(cd, res)
			Expect(actualMatch).To(Equal(true))
		})

		ginkgo.It("should not match if access type is not in include list", func() {
			cd := cdv2.ComponentDescriptor{}
			res := cdv2.Resource{
				Access: cdv2.NewEmptyUnstructured(cdv2.OCIRegistryType),
			}
			spec := filter.AccessTypeFilterSpec{
				IncludeAccessTypes: []string{
					cdv2.LocalOCIBlobType,
				},
			}

			f, err := filter.NewAccessTypeFilter(spec)
			Expect(err).ToNot(HaveOccurred())

			actualMatch := f.Matches(cd, res)
			Expect(actualMatch).To(Equal(false))
		})

		ginkgo.It("should return error upon creation if include list is empty", func() {
			spec := filter.AccessTypeFilterSpec{
				IncludeAccessTypes: []string{},
			}
			_, err := filter.NewAccessTypeFilter(spec)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("includeAccessTypes must not be empty"))
		})

	})

	ginkgo.Context("resourceTypeFilter", func() {

		ginkgo.It("should match if resource type is in include list", func() {
			cd := cdv2.ComponentDescriptor{}
			res := cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-res",
					Version: "v0.1.0",
					Type:    cdv2.OCIImageType,
				},
			}
			spec := filter.ResourceTypeFilterSpec{
				IncludeResourceTypes: []string{
					cdv2.OCIImageType,
				},
			}

			f, err := filter.NewResourceTypeFilter(spec)
			Expect(err).ToNot(HaveOccurred())

			actualMatch := f.Matches(cd, res)
			Expect(actualMatch).To(Equal(true))
		})

		ginkgo.It("should not match if resource type is not in include list", func() {
			cd := cdv2.ComponentDescriptor{}
			res := cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-res",
					Version: "v0.1.0",
					Type:    "helm",
				},
			}
			spec := filter.ResourceTypeFilterSpec{
				IncludeResourceTypes: []string{
					cdv2.OCIImageType,
				},
			}

			f, err := filter.NewResourceTypeFilter(spec)
			Expect(err).ToNot(HaveOccurred())

			actualMatch := f.Matches(cd, res)
			Expect(actualMatch).To(Equal(false))
		})

		ginkgo.It("should return error upon creation if include list is empty", func() {
			spec := filter.ResourceTypeFilterSpec{
				IncludeResourceTypes: []string{},
			}
			_, err := filter.NewResourceTypeFilter(spec)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("includeResourceTypes must not be empty"))
		})

	})

	ginkgo.Context("componentNameFilter", func() {

		ginkgo.It("should match if component name is in include list", func() {
			cd := cdv2.ComponentDescriptor{
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name: "github.com/test/my-component",
					},
				},
			}
			res := cdv2.Resource{}
			spec := filter.ComponentNameFilterSpec{
				IncludeComponentNames: []string{
					"github.com/test/my-component",
				},
			}

			f1, err := filter.NewComponentNameFilter(spec)
			Expect(err).ToNot(HaveOccurred())

			match1 := f1.Matches(cd, res)
			Expect(match1).To(Equal(true))

			spec = filter.ComponentNameFilterSpec{
				IncludeComponentNames: []string{
					"github.com/test/*",
				},
			}
			f2, err := filter.NewComponentNameFilter(spec)
			Expect(err).ToNot(HaveOccurred())

			match2 := f2.Matches(cd, res)
			Expect(match2).To(Equal(true))
		})

		ginkgo.It("should not match if component name is not in include list", func() {
			cd := cdv2.ComponentDescriptor{
				ComponentSpec: cdv2.ComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name: "github.com/test/my-component",
					},
				},
			}
			res := cdv2.Resource{}
			spec := filter.ComponentNameFilterSpec{
				IncludeComponentNames: []string{
					"github.com/test/my-other-component",
				},
			}

			f1, err := filter.NewComponentNameFilter(spec)
			Expect(err).ToNot(HaveOccurred())

			match1 := f1.Matches(cd, res)
			Expect(match1).To(Equal(false))

			spec = filter.ComponentNameFilterSpec{
				IncludeComponentNames: []string{
					"github.com/test-2/*",
				},
			}
			f2, err := filter.NewComponentNameFilter(spec)
			Expect(err).ToNot(HaveOccurred())

			match2 := f2.Matches(cd, res)
			Expect(match2).To(Equal(false))
		})

		ginkgo.It("should return error upon creation if include list is empty", func() {
			spec := filter.ComponentNameFilterSpec{
				IncludeComponentNames: []string{},
			}
			_, err := filter.NewComponentNameFilter(spec)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("includeComponentNames must not be empty"))
		})

		ginkgo.It("should return error upon creation if regexp is invalid", func() {
			spec := filter.ComponentNameFilterSpec{
				IncludeComponentNames: []string{
					"github.com/\\",
				},
			}
			_, err := filter.NewComponentNameFilter(spec)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error parsing regexp"))
		})

	})

})
