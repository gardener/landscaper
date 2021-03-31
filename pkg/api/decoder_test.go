// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/api/testdata"
	v1 "github.com/gardener/landscaper/pkg/api/testdata/v1"
	v2 "github.com/gardener/landscaper/pkg/api/testdata/v2"
)

var _ = Describe("UniversalInternalVersionDecoder", func() {

	scheme := runtime.NewScheme()

	BeforeEach(func() {
		Expect(testdata.AddToScheme(scheme)).To(Succeed())
		Expect(v1.AddToScheme(scheme)).To(Succeed())
		Expect(v2.AddToScheme(scheme)).To(Succeed())
	})

	It("should automatically convert version v1 to v2 using the internal version", func() {
		data := `
apiVersion: somegroup.gardener.cloud/v1
kind: SomeType
numberString: "2"
`

		res := &v2.SomeType{}
		_, _, err := api.NewDecoder(scheme).Decode([]byte(data), nil, res)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Number).To(Equal(2))
	})

	It("should automatically convert version v2 to v1 using the internal version", func() {
		data := `
apiVersion: somegroup.gardener.cloud/v2
kind: SomeType
number: 2
`

		res := &v1.SomeType{}
		_, _, err := api.NewDecoder(scheme).Decode([]byte(data), nil, res)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.NumberString).To(Equal("2"))
	})
})
