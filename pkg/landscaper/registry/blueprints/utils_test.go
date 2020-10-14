// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprintsregistry_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	blueprintsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/blueprints"
)

var _ = Describe("Utils", func() {

	Context("ParseDefinitionRef", func() {
		It("should successfully parse a reference into its name and version", func() {
			vn, err := blueprintsregistry.ParseDefinitionRef("my-comp:1.0.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(vn.Name).To(Equal("my-comp"))
			Expect(vn.Version).To(Equal("1.0.0"))
		})

		It("should successfully parse a reference with a complete url into its name and version", func() {
			vn, err := blueprintsregistry.ParseDefinitionRef("my.host/my-comp:1.0.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(vn.Name).To(Equal("my.host/my-comp"))
			Expect(vn.Version).To(Equal("1.0.0"))
		})

		It("should successfully parse a reference with a complete url and specific port into its name and version", func() {
			vn, err := blueprintsregistry.ParseDefinitionRef("my.host:5000/my-comp:1.0.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(vn.Name).To(Equal("my.host:5000/my-comp"))
			Expect(vn.Version).To(Equal("1.0.0"))
		})

		It("should successfully parse a reference into its name and version with a colon", func() {
			vn, err := blueprintsregistry.ParseDefinitionRef("my:comp:1.0.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(vn.Name).To(Equal("my:comp"))
			Expect(vn.Version).To(Equal("1.0.0"))
		})

		It("should throw a version not parsable error when an error occures", func() {
			_, err := blueprintsregistry.ParseDefinitionRef("my-comp-1.0.0")
			Expect(err).To(HaveOccurred())
		})
	})
})
