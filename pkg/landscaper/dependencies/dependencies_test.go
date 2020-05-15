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

package dependencies_test

import (
	"github.com/gardener/landscaper/pkg/landscaper/component"
	"github.com/gardener/landscaper/pkg/landscaper/dependencies"
	"github.com/gardener/landscaper/test/utils"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestTestDefinition(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dependencies Suite")
}

var _ = Describe("Import Dependencies tests", func() {
	It("imports should be satisfied", func() {
		rawComponent, err := utils.ReadComponentFromFile("./testdata/01-component.yaml")
		Expect(err).ToNot(HaveOccurred())
		testComponent, err := component.New(rawComponent)
		Expect(err).ToNot(HaveOccurred())

		rawComponent2, err := utils.ReadComponentFromFile("./testdata/02-component.yaml")
		Expect(err).ToNot(HaveOccurred())
		testComponent2, err := component.New(rawComponent2)
		Expect(err).ToNot(HaveOccurred())

		err = dependencies.CheckImportSatisfaction(testComponent, []*component.Component{testComponent2}, nil)
		Expect(err).ToNot(HaveOccurred())
	})

	It("imports should be satisfied when no imports are required", func() {
		rawComponent, err := utils.ReadComponentFromFile("./testdata/02-component.yaml")
		Expect(err).ToNot(HaveOccurred())
		testComponent, err := component.New(rawComponent)
		Expect(err).ToNot(HaveOccurred())

		err = dependencies.CheckImportSatisfaction(testComponent, []*component.Component{}, nil)
		Expect(err).ToNot(HaveOccurred())
	})

	It("imports should not be satisfied", func() {
		rawComponent, err := utils.ReadComponentFromFile("./testdata/01-component.yaml")
		Expect(err).ToNot(HaveOccurred())
		testComponent, err := component.New(rawComponent)
		Expect(err).ToNot(HaveOccurred())

		rawComponent3, err := utils.ReadComponentFromFile("./testdata/03-component.yaml")
		Expect(err).ToNot(HaveOccurred())
		testComponent3, err := component.New(rawComponent3)
		Expect(err).ToNot(HaveOccurred())

		err = dependencies.CheckImportSatisfaction(testComponent, []*component.Component{testComponent3}, nil)
		Expect(err).To(HaveOccurred())
	})

	It("imports should be satisfied from landscape config", func() {
		rawComponent, err := utils.ReadComponentFromFile("./testdata/01-component.yaml")
		Expect(err).ToNot(HaveOccurred())
		testComponent, err := component.New(rawComponent)
		Expect(err).ToNot(HaveOccurred())

		rawComponent3, err := utils.ReadComponentFromFile("./testdata/03-component.yaml")
		Expect(err).ToNot(HaveOccurred())
		testComponent3, err := component.New(rawComponent3)
		Expect(err).ToNot(HaveOccurred())

		err = dependencies.CheckImportSatisfaction(testComponent, []*component.Component{testComponent3}, map[string]interface{}{
			"test": map[string]interface{}{
				"value1": true,
			},
		})
		Expect(err).ToNot(HaveOccurred())
	})
})
