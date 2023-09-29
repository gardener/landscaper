// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package compression

import (
	"bytes"
	"io"
)

type MatchReader interface {
	io.Reader
	Reset()
}

type matchReader struct {
	read    []byte
	buffer  *bytes.Buffer
	reader  io.Reader
	current io.Reader
}

var _ MatchReader = (*matchReader)(nil)

func NewMatchReader(r io.Reader) *matchReader {
	return &matchReader{
		buffer:  bytes.NewBuffer(nil),
		reader:  r,
		current: r,
	}
}

func (r *matchReader) Read(buf []byte) (int, error) {
	n, err := r.current.Read(buf)
	if n > 0 {
		_, err = r.buffer.Write(buf[:n])
	}
	return n, err
}

func (r *matchReader) Reset() {
	if r.buffer.Len() > 0 {
		if r.buffer.Len() > len(r.read) {
			r.read = r.buffer.Bytes()
		}
		r.buffer = bytes.NewBuffer(nil)
		r.current = io.MultiReader(bytes.NewBuffer(r.read), r.reader)
	}
}

func (r *matchReader) Reader() io.Reader {
	r.Reset()
	r.reader = nil
	return r.current
}
