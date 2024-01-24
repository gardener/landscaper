// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package iotools

import (
	"io"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/generics"
)

////////////////////////////////////////////////////////////////////////////////

type additionalCloser[T any] struct {
	msg              []interface{}
	wrapped          T
	additionalCloser io.Closer
}

func (c *additionalCloser[T]) Close() error {
	var list *errors.ErrorList
	if len(c.msg) == 0 {
		list = errors.ErrListf("close")
	} else {
		if s, ok := c.msg[0].(string); ok && len(c.msg) > 1 {
			list = errors.ErrListf(s, c.msg[1:]...)
		} else {
			list = errors.ErrList(c.msg...)
		}
	}
	if cl, ok := generics.TryCast[io.Closer](c.wrapped); ok {
		list.Add(cl.Close())
	}
	if c.additionalCloser != nil {
		list.Add(c.additionalCloser.Close())
	}
	return list.Result()
}

func newAdditionalCloser[T any](w T, closer io.Closer, msg ...interface{}) additionalCloser[T] {
	return additionalCloser[T]{
		wrapped:          w,
		msg:              msg,
		additionalCloser: closer,
	}
}

////////////////////////////////////////////////////////////////////////////////

type readCloser struct {
	additionalCloser[io.Reader]
}

var _ io.ReadCloser = (*readCloser)(nil)

// Deprecated: use AddReaderCloser .
func AddCloser(reader io.ReadCloser, closer io.Closer, msg ...string) io.ReadCloser {
	return AddReaderCloser(reader, closer, generics.ConvertSliceTo[any](msg)...)
}

func ReadCloser(r io.Reader) io.ReadCloser {
	return AddReaderCloser(r, nil)
}

func AddReaderCloser(reader io.Reader, closer io.Closer, msg ...interface{}) io.ReadCloser {
	return &readCloser{
		additionalCloser: newAdditionalCloser[io.Reader](reader, closer, msg...),
	}
}

func (c *readCloser) Read(p []byte) (n int, err error) {
	return c.wrapped.Read(p)
}

type writeCloser struct {
	additionalCloser[io.Writer]
}

var _ io.WriteCloser = (*writeCloser)(nil)

func WriteCloser(w io.Writer) io.WriteCloser {
	return AddWriterCloser(w, nil)
}

func AddWriterCloser(writer io.Writer, closer io.Closer, msg ...interface{}) io.WriteCloser {
	return &writeCloser{
		additionalCloser: newAdditionalCloser[io.Writer](writer, closer, msg...),
	}
}

func (c *writeCloser) Write(p []byte) (n int, err error) {
	return c.wrapped.Write(p)
}
