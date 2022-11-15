// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Queue Test Suite")
}

var _ = Describe("Queue Implementation Tests", func() {

	var (
		q    Queue[string]
		data []string
	)

	BeforeEach(func() {
		data = []string{"a", "b", "c"}
		q = New(data...)
	})

	Context("NewString", func() {
		It("should create a new queue from a given slice", func() {
			Expect(q.elems).To(Equal(data))
		})
	})

	Context("Pop", func() {
		It("should return and remove the first element of the queue", func() {
			elem, err := q.Pop()
			Expect(err).ToNot(HaveOccurred())
			Expect(elem).To(Equal(data[0]))
			data = data[1:]
			Expect(q.elems).To(Equal(data))
			elem, err = q.Pop()
			Expect(err).ToNot(HaveOccurred())
			Expect(elem).To(Equal(data[0]))
			data = data[1:]
			Expect(q.elems).To(Equal(data))
			elem, err = q.Pop()
			Expect(err).ToNot(HaveOccurred())
			Expect(elem).To(Equal(data[0]))
			data = data[1:]
			Expect(q.elems).To(Equal(data))
			_, err = q.Pop()
			Expect(err).To(HaveOccurred())
			Expect(q.Len()).To(Equal(0))
		})
	})

	Context("Peek", func() {
		It("should return the first element of the queue without removing it", func() {
			elem, err := q.Peek()
			Expect(err).ToNot(HaveOccurred())
			Expect(elem).To(Equal(data[0]))
			Expect(q.elems).To(Equal(data))
		})
	})

	Context("Len", func() {
		It("should return the length of the queue", func() {
			Expect(q.Len()).To(Equal(len(q.elems)))
		})
	})

	Context("Append", func() {
		It("should append new elements", func() {
			addData := []string{"d", "e", "f"}
			q.Append(addData...)
			newData := append(data, addData...)
			Expect(q.elems).To(Equal(newData))

			// ensure that data as well as addData have not been modified
			Expect(data).To(HaveLen(3))
			Expect(addData).To(HaveLen(3))
		})
	})

	Context("Copy", func() {
		It("should return an independent copy", func() {
			q2 := q.Copy()
			Expect(q2.elems).To(Equal(q.elems))
			elem, err := q2.Pop()
			Expect(err).ToNot(HaveOccurred())
			Expect(elem).To(Equal(q.elems[0]))
			Expect(q2.elems).To(Equal(q.elems[1:]))
		})
	})

})
