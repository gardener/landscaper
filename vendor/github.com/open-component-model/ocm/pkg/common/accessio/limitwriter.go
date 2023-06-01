// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessio

import (
	"bytes"
	"io"
)

const DESCRIPTOR_LIMIT = int64(8196 * 4)

// LimitWriter returns a Writer that writes to w
// but stops with EOF after n bytes.
// The underlying implementation is a *LimitedWriter.
func LimitWriter(w io.Writer, n int64) io.Writer { return &LimitedWriter{w, n} }

// A LimitedWriter writes to W but limits the amount of
// data written to just N bytes. Each call to Write
// updates N to reflect the new amount remaining.
// Write returns EOF when N <= 0 or when the underlying W returns EOF.
type LimitedWriter struct {
	W io.Writer // underlying reader
	N int64     // max bytes remaining
}

func (l *LimitedWriter) Write(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > l.N {
		p = p[0:l.N]
	}
	n, err = l.W.Write(p)
	l.N -= int64(n)
	return
}

func LimitBuffer(n int64) *LimitedBuffer {
	buf := &LimitedBuffer{max: n}
	buf.LimitedWriter = &LimitedWriter{&buf.buffer, n + 1}
	return buf
}

type LimitedBuffer struct {
	*LimitedWriter
	max    int64
	buffer bytes.Buffer
}

func (b *LimitedBuffer) Exceeded() bool {
	return b.LimitedWriter.N > b.max
}

func (b *LimitedBuffer) Bytes() []byte {
	return b.buffer.Bytes()
}
