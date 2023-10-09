// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package registries

import (
	"github.com/gardener/landscaper/pkg/components/cnudie"
	"github.com/gardener/landscaper/pkg/components/ocmlib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
	It("useOCMLib: default (false) / useOCM: default (false)", func() {
		_, ok := GetFactory().(*cnudie.Factory)
		Expect(ok).To(BeTrue())
	})
})
