// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cdutils

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
)

var _ = Describe("resource utils", func() {

	Context("#MergeIdentityObjectMeta", func() {

		It("should merge labels", func() {
			meta1 := cdv2.IdentityObjectMeta{
				Name: "test",
				Labels: cdv2.Labels{
					{Name: "k2", Value: []byte("v2")},
					{Name: "k3", Value: []byte("xx")},
				},
			}
			meta2 := cdv2.IdentityObjectMeta{
				Name: "test",
				Labels: cdv2.Labels{
					{Name: "k1", Value: []byte("v1")},
					{Name: "k3", Value: []byte("v3")},
					{Name: "k4", Value: []byte("v4")},
				},
			}

			result := MergeIdentityObjectMeta(meta1, meta2)
			Expect(result.Labels).To(ConsistOf(
				cdv2.Label{Name: "k1", Value: []byte("v1")},
				cdv2.Label{Name: "k2", Value: []byte("v2")},
				cdv2.Label{Name: "k3", Value: []byte("v3")},
				cdv2.Label{Name: "k4", Value: []byte("v4")},
			))
		})
	})
})
