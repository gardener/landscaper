// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessio

import (
	"io"
)

type CountingReader struct {
	reader io.Reader
	count  int64
}

func (r *CountingReader) Size() int64 {
	return r.count
}

func (r *CountingReader) Read(buf []byte) (int, error) {
	c, err := r.reader.Read(buf)
	r.count += int64(c)
	return c, err
}

func NewCountingReader(r io.Reader) *CountingReader {
	return &CountingReader{
		reader: r,
		count:  0,
	}
}

////////////////////////////////////////////////////////////////////////////////

type CountingWriter struct {
	writer io.Writer
	count  int64
}

func (w *CountingWriter) Size() int64 {
	return w.count
}

func (w *CountingWriter) Write(buf []byte) (int, error) {
	c, err := w.writer.Write(buf)
	w.count += int64(c)
	return c, err
}

func NewCountingWriter(w io.Writer) *CountingWriter {
	return &CountingWriter{
		writer: w,
		count:  0,
	}
}
