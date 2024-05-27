// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package registries

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/pkg/components/cnudie"
	"github.com/gardener/landscaper/pkg/components/ocmlib"
)

var _ = Describe("check correct factory instantiation all condition variants", func() {
	BeforeEach(func() {
		ocmLibraryMode = nil
	})
	// Standard Cases
	It("useOCMLib: false / useOCM: false", func() {
		SetOCMLibraryMode(false)
		_, ok := GetFactory(false).(*cnudie.Factory)
		Expect(ok).To(BeTrue())
	})
	It("useOCMLib: false / useOCM: true", func() {
		SetOCMLibraryMode(false)
		_, ok := GetFactory(true).(*ocmlib.Factory)
		Expect(ok).To(BeTrue())
	})
	It("useOCMLib: true / useOCM: false", func() {
		SetOCMLibraryMode(false)
		_, ok := GetFactory(true).(*ocmlib.Factory)
		Expect(ok).To(BeTrue())
	})
	It("useOCMLib: true / useOCM: true", func() {
		SetOCMLibraryMode(true)
		_, ok := GetFactory(true).(*ocmlib.Factory)
		Expect(ok).To(BeTrue())
	})

	// Default Case
	It("useOCMLib: default (true) / useOCM: default (true)", func() {
		_, ok := GetFactory().(*ocmlib.Factory)
		Expect(ok).To(BeTrue())
	})
})
