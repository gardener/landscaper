// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
