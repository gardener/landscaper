// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessio

import (
	"io"

	"github.com/opencontainers/go-digest"
)

type DigestReader struct {
	reader   io.Reader
	digester digest.Digester
	count    int64
}

func (r *DigestReader) Size() int64 {
	return r.count
}

func (r *DigestReader) Digest() digest.Digest {
	return r.digester.Digest()
}

func (r *DigestReader) Read(buf []byte) (int, error) {
	c, err := r.reader.Read(buf)
	if c > 0 {
		r.count += int64(c)
		r.digester.Hash().Write(buf[:c])
	}
	return c, err
}

func NewDefaultDigestReader(r io.Reader) *DigestReader {
	return &DigestReader{
		reader:   r,
		digester: digest.Canonical.Digester(),
		count:    0,
	}
}

func NewDigestReaderWith(algorithm digest.Algorithm, r io.Reader) *DigestReader {
	return &DigestReader{
		reader:   r,
		digester: algorithm.Digester(),
		count:    0,
	}
}

func Digest(access DataAccess) (digest.Digest, error) {
	reader, err := access.Reader()
	if err != nil {
		return "", err
	}
	defer reader.Close()

	dig, err := digest.FromReader(reader)
	if err != nil {
		return "", err
	}
	return dig, nil
}
