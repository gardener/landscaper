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

package registry_test

import (
	"fmt"
	"github.com/gardener/landscaper/pkg/landscaper/registry"
	"github.com/go-logr/logr/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	localTestData1 = "./testdata/local-1"
	localTestData2 = "./testdata/local-2"
)

var _ = Describe("Local Registry", func() {

	Context("initialize index", func() {
		It("should be successfully initialized with one path", func() {
			_, err := registry.NewLocalRegistry(testing.NullLogger{}, []string{localTestData1})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should be successfully initialized with multiple paths", func() {
			_, err := registry.NewLocalRegistry(testing.NullLogger{}, []string{
				localTestData1,
				localTestData2,
			})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should be successfully initialized with multiple paths that are subpaths", func() {
			_, err := registry.NewLocalRegistry(testing.NullLogger{}, []string{
				localTestData1,
				fmt.Sprintf("%s/comp1", localTestData1),
			})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("GetDefinition", func() {

		var reg registry.Registry

		BeforeEach(func() {
			var err error
			reg, err = registry.NewLocalRegistry(testing.NullLogger{}, []string{localTestData1})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return a component by name", func() {
			_, err := reg.GetDefinition("root-definition", "1.0.0")
			Expect(err).ToNot(HaveOccurred())

			_, err = reg.GetDefinition("sub-definition-1", "1.1.0")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return an error if the name is incorrect", func() {
			_, err := reg.GetDefinition("unkown-definition", "1.0.0")
			Expect(registry.IsComponentNotFoundError(err)).To(BeTrue())
		})

		It("should return an error if the version is incorrect", func() {
			_, err := reg.GetDefinition("sub-definition-1", "1.0.0")
			Expect(registry.IsVersionNotFoundError(err)).To(BeTrue())
		})
	})

	Context("GetVersions", func() {

		var reg registry.Registry

		BeforeEach(func() {
			var err error
			reg, err = registry.NewLocalRegistry(testing.NullLogger{}, []string{localTestData1})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return one available version of a component", func() {
			versions, err := reg.GetVersions("root-definition")
			Expect(err).ToNot(HaveOccurred())
			Expect(versions).To(ConsistOf("1.0.0"))
		})

		It("should return multiple available versions of a component", func() {
			versions, err := reg.GetVersions("sub-definition-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(versions).To(ConsistOf("1.1.0", "1.2.0"))
		})

		It("should return an error if the name is incorrect", func() {
			_, err := reg.GetVersions("unkown-definition")
			Expect(registry.IsComponentNotFoundError(err)).To(BeTrue())
		})

	})

})
