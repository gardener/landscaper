// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessio

import (
	"io"

	"github.com/opencontainers/go-digest"
)

type writer io.WriteCloser

type DigestWriter struct {
	writer
	digester digest.Digester
	count    int64
}

func (r *DigestWriter) Size() int64 {
	return r.count
}

func (r *DigestWriter) Digest() digest.Digest {
	return r.digester.Digest()
}

func (r *DigestWriter) Write(buf []byte) (int, error) {
	c, err := r.writer.Write(buf)
	if c > 0 {
		r.count += int64(c)
		r.digester.Hash().Write(buf[:c])
	}
	return c, err
}

func NewDefaultDigestWriter(w io.WriteCloser) *DigestWriter {
	return &DigestWriter{
		writer:   w,
		digester: digest.Canonical.Digester(),
		count:    0,
	}
}

func NewDigestWriterWith(algorithm digest.Algorithm, w io.WriteCloser) *DigestWriter {
	return &DigestWriter{
		writer:   w,
		digester: algorithm.Digester(),
		count:    0,
	}
}
