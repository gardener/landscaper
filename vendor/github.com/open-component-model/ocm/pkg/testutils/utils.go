// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package testutils

import (
	"fmt"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/types"

	"github.com/open-component-model/ocm/pkg/utils"
)

func Close(c io.Closer, msg ...interface{}) {
	Defer(c.Close, msg...)
}

func Defer(f func() error, msg ...interface{}) {
	DeferWithOffset(1, f, msg...)
}

func DeferWithOffset(o int, f func() error, msg ...interface{}) {
	err := f()
	if err != nil {
		switch len(msg) {
		case 0:
			ExpectWithOffset(1+o, err).To(Succeed())
		case 1:
			Fail(fmt.Sprintf("%s: %s", msg[0], err), 1+o)
		default:
			Fail(fmt.Sprintf("%s: %s", fmt.Sprintf(msg[0].(string), msg[1:]...), err), 1+o)
		}
	}
}

func Must[T any](o T, err error) T {
	ExpectWithOffset(1, err).To(Succeed())
	return o
}

func Must2[T any, V any](a T, b V, err error) (T, V) {
	ExpectWithOffset(1, err).To(Succeed())
	return a, b
}

type result[T any] struct {
	res T
	err error
}

func (r result[T]) Must(offset ...int) T {
	ExpectWithOffset(utils.Optional(offset...)+1, r.err).To(Succeed())
	return r.res
}

func R[T any](o T, err error) result[T] {
	return Calling(o, err)
}

func Calling[T any](o T, err error) result[T] {
	return result[T]{o, err}
}

func MustWithOffset[T any](offset int, res result[T]) T {
	ExpectWithOffset(offset+1, res.err).To(Succeed())
	return res.res
}

func MustBeNonNil[T any](o T) T {
	ExpectWithOffset(1, o).NotTo(BeNil())
	return o
}

func MustBeSuccessful(err error) {
	ExpectWithOffset(1, err).To(Succeed())
}

func MustBeSuccessfulWithOffset(offset int, err error) {
	ExpectWithOffset(offset+1, err).To(Succeed())
}

func MustFailWithMessage(err error, msg string) {
	ExpectWithOffset(1, err).NotTo(BeNil())
	ExpectWithOffset(1, err.Error()).To(Equal(msg))
}

func ExpectError(values ...interface{}) types.Assertion {
	return Expect(values[len(values)-1])
}
