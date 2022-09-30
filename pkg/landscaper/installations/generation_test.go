// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
)

var _ = Describe("config generation", func() {

	var (
		inst *lsv1alpha1.Installation
	)

	BeforeEach(func() {
		inst = &lsv1alpha1.Installation{}
		inst.Generation = 5
		inst.Status.Imports = []lsv1alpha1.ImportStatus{
			{
				Name:             "a",
				ConfigGeneration: "abc",
			},
			{
				Name:             "b",
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

		inst.Status.Imports = []lsv1alpha1.ImportStatus{
			{
				Name:             "b",
				ConfigGeneration: "def",
			},
			{
				Name:             "a",
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

		inst.Status.Imports = []lsv1alpha1.ImportStatus{
			{
				Name:             "b",
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
