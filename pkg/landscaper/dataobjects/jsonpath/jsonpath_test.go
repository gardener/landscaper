// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package jsonpath_test

import (
	"testing"

	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestTestDefinition(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "JSONPath Suite")
}

var _ = Describe("JSONPath", func() {

	Context("GetValue", func() {

		var (
			data = map[string]interface{}{
				"level10": map[string]interface{}{
					"key1": true,
					"key2": 10,
				},
				"level11": map[string]interface{}{
					"level20": map[string]interface{}{
						"key1": "val",
					},
				},
				"level12": map[string]interface{}{
					"key1": "{ \"nested\": true }",
					"key2": map[string]interface{}{
						"nested": true,
					},
				},
			}
		)

		It("should returns the value of a nested struct", func() {
			var val int
			err := jsonpath.GetValue(".level10.key2", data, &val)
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(Equal(10))
		})

		It("should throw an error if the value is of wrong type", func() {
			var val bool
			err := jsonpath.GetValue(".level11.level20.key1", data, &val)
			Expect(err).To(HaveOccurred())
		})

		It("should return a string even if in the string is a valid json or yaml struct", func() {
			var val string
			err := jsonpath.GetValue(".level12.key1", data, &val)
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(BeAssignableToTypeOf(""))
		})

		It("should return a sub struct", func() {
			var val map[string]interface{}
			err := jsonpath.GetValue(".level12.key2", data, &val)
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(BeAssignableToTypeOf(map[string]interface{}{}))
			Expect(val["nested"]).To(Equal(true))
		})

	})

	Context("construct", func() {
		It("should construct a struct based on a jsonpath with depth of 2", func() {
			res, err := jsonpath.Construct(".key1.key2", "val1")
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveKey("key1"))
			Expect(res["key1"]).To(HaveKey("key2"))
		})

		It("should construct a struct based on a jsonpath and set the provided value as the last element", func() {
			var (
				text = ".key1.key2"
				val  = "val1"
			)
			res, err := jsonpath.Construct(text, val)
			Expect(err).ToNot(HaveOccurred())

			var resVal interface{}
			err = jsonpath.GetValue(text, res, &resVal)
			Expect(err).ToNot(HaveOccurred())

			Expect(resVal).To(Equal(val))
		})
	})

})
