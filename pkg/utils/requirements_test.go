// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"errors"

	"github.com/gardener/landscaper/pkg/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type staticCounter struct {
	c int
}

func (c *staticCounter) increment() {
	c.c += 1
}
func (c *staticCounter) count() int {
	return c.c
}

var _ = Describe("Requirements", func() {

	var (
		req     utils.Requirements
		incFunc func() error
		errFunc func() error
		count   func() int
	)

	const (
		incReq = "inc"
		errReq = "err"
	)

	BeforeEach(func() {
		req = utils.NewRequirements()

		counter := &staticCounter{}
		incFunc = func() error {
			counter.increment()
			return nil
		}
		errFunc = func() error {
			return errors.New("test error")
		}
		count = func() int {
			return counter.count()
		}
		req.Register(incReq, incFunc)
		req.Register(errReq, errFunc)
	})

	Context("MetaTest", func() {
		It("should increment the counter with each call", func() {
			Expect(count()).To(Equal(0))
			Expect(incFunc()).To(Succeed())
			Expect(count()).To(Equal(1))
			Expect(incFunc()).To(Succeed())
			Expect(count()).To(Equal(2))
		})
	})

	Context("RequirementSatisfaction", func() {

		It("should call satisfy exactly once if not satisfied", func() {
			Expect(count()).To(Equal(0))
			Expect(req.Require(incReq)).To(Succeed())
			Expect(count()).To(Equal(1))
			Expect(req.IsSatisfied(incReq)).To(BeTrue())
			Expect(req.Require(incReq)).To(Succeed())
			Expect(count()).To(Equal(1))
		})

		It("should not satisfy the requirement if the satisfy function returns an error", func() {
			Expect(req.Require(errReq)).ToNot(Succeed())
			Expect(req.IsSatisfied(errReq)).To(BeFalse())
		})

	})
})
