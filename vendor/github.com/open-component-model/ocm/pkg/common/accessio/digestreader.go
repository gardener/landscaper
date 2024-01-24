// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessio

import (
	"crypto"
	"io"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/iotools"
)

// Deprecated: use iotools.DigestReader.
type DigestReader = iotools.DigestReader

// Deprecated: use iotools.NewDefaultDigestReader.
func NewDefaultDigestReader(r io.Reader) *iotools.DigestReader {
	return iotools.NewDigestReaderWith(digest.Canonical, r)
}

// Deprecated: use iotools.NewDigestReaderWith.
func NewDigestReaderWith(algorithm digest.Algorithm, r io.Reader) *iotools.DigestReader {
	return iotools.NewDigestReaderWith(algorithm, r)
}

// Deprecated: use iotools.NewDigestReaderWithHash.
func NewDigestReaderWithHash(hash crypto.Hash, r io.Reader) *iotools.DigestReader {
	return iotools.NewDigestReaderWithHash(hash, r)
}

// Deprecated: use iotools.VerifyingReader.
func VerifyingReader(r io.ReadCloser, digest digest.Digest) io.ReadCloser {
	return iotools.VerifyingReader(r, digest)
}

// Deprecated: use iotools.VerifyingReaderWithHash.
func VerifyingReaderWithHash(r io.ReadCloser, hash crypto.Hash, digest string) io.ReadCloser {
	return iotools.VerifyingReaderWithHash(r, hash, digest)
}

// Deprecated: use blobaccess.Digest.
func Digest(access blobaccess.DataAccess) (digest.Digest, error) {
	return blobaccess.Digest(access)
}
