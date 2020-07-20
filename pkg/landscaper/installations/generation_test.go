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

package installations_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
)

var _ = Describe("config generation", func() {

	var (
		inst *lsv1alpha1.Installation
	)

	BeforeEach(func() {
		inst = &lsv1alpha1.Installation{}
		inst.Generation = 5
		inst.Status.Imports = []lsv1alpha1.ImportState{
			{
				To:               "a",
				ConfigGeneration: "abc",
			},
			{
				To:               "b",
				ConfigGeneration: "def",
			},
		}
	})

	It("should create the same hash when invoked multiple times", func() {
		gen, err := installations.CreateGenerationHash(inst)
		Expect(err).ToNot(HaveOccurred())

		gen2, err := installations.CreateGenerationHash(inst)
		Expect(err).ToNot(HaveOccurred())

		Expect(gen).To(Equal(gen2))
	})

	It("should create the same hash even when the import states order changes", func() {
		gen, err := installations.CreateGenerationHash(inst)
		Expect(err).ToNot(HaveOccurred())

		inst.Status.Imports = []lsv1alpha1.ImportState{
			{
				To:               "b",
				ConfigGeneration: "def",
			},
			{
				To:               "a",
				ConfigGeneration: "abc",
			},
		}

		gen2, err := installations.CreateGenerationHash(inst)
		Expect(err).ToNot(HaveOccurred())

		Expect(gen).To(Equal(gen2))
	})

	It("should create a new hash when the generation changes", func() {
		gen, err := installations.CreateGenerationHash(inst)
		Expect(err).ToNot(HaveOccurred())

		inst.Generation = 7

		gen2, err := installations.CreateGenerationHash(inst)
		Expect(err).ToNot(HaveOccurred())

		Expect(gen).ToNot(Equal(gen2))
	})

	It("should create a new hash when the import state changes", func() {
		gen, err := installations.CreateGenerationHash(inst)
		Expect(err).ToNot(HaveOccurred())

		inst.Status.Imports = []lsv1alpha1.ImportState{
			{
				To:               "b",
				ConfigGeneration: "def",
			},
		}

		gen2, err := installations.CreateGenerationHash(inst)
		Expect(err).ToNot(HaveOccurred())

		Expect(gen).ToNot(Equal(gen2))
	})

	It("should create a new hash when the import state config generation changes", func() {
		gen, err := installations.CreateGenerationHash(inst)
		Expect(err).ToNot(HaveOccurred())

		inst.Status.Imports[0].ConfigGeneration = "xyz"

		gen2, err := installations.CreateGenerationHash(inst)
		Expect(err).ToNot(HaveOccurred())

		Expect(gen).ToNot(Equal(gen2))
	})

})
