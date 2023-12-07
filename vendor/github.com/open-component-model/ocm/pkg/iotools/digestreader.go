// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package iotools

import (
	"crypto"
	"hash"
	"io"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/errors"
)

// wow. digest does support a map with supported digesters. Unfortunately this one does not
// contain all the crypto hashes AND this map is private AND there is no function to add entries,
// so that it cannot be extended from outside the package.
// Therefore, we have to fake it a little to support digests with other crypto hashes.

type DigestReader struct {
	reader io.Reader
	alg    digest.Algorithm
	hash   hash.Hash
	count  int64
}

func (r *DigestReader) Size() int64 {
	return r.count
}

func (r *DigestReader) Digest() digest.Digest {
	return digest.NewDigest(r.alg, r.hash)
}

func (r *DigestReader) Read(buf []byte) (int, error) {
	c, err := r.reader.Read(buf)
	if c > 0 {
		r.count += int64(c)
		r.hash.Write(buf[:c])
	}
	return c, err
}

func NewDefaultDigestReader(r io.Reader) *DigestReader {
	return NewDigestReaderWith(digest.Canonical, r)
}

func NewDigestReaderWith(algorithm digest.Algorithm, r io.Reader) *DigestReader {
	digester := algorithm.Digester()
	return &DigestReader{
		reader: r,
		hash:   digester.Hash(),
		alg:    algorithm,
		count:  0,
	}
}

func NewDigestReaderWithHash(hash crypto.Hash, r io.Reader) *DigestReader {
	return &DigestReader{
		reader: r,
		hash:   hash.New(),
		alg:    digest.Algorithm(hash.String()), // fake a non-supported digest algorithm
		count:  0,
	}
}

type verifiedReader struct {
	closer io.Closer
	*DigestReader
	hash   string
	digest string
}

func (v *verifiedReader) Close() error {
	err := v.closer.Close()
	if err != nil {
		return err
	}
	dig := v.DigestReader.Digest()
	if dig.Hex() != v.digest {
		return errors.Newf("%s digest mismatch: expected %s, found %s", v.hash, v.digest, dig.Hex())
	}
	return nil
}

func VerifyingReader(r io.ReadCloser, digest digest.Digest) io.ReadCloser {
	return &verifiedReader{
		closer:       r,
		DigestReader: NewDigestReaderWith(digest.Algorithm(), r),
		hash:         digest.Algorithm().String(),
		digest:       digest.Hex(),
	}
}

func VerifyingReaderWithHash(r io.ReadCloser, hash crypto.Hash, digest string) io.ReadCloser {
	return &verifiedReader{
		closer:       r,
		DigestReader: NewDigestReaderWithHash(hash, r),
		hash:         hash.String(),
		digest:       digest,
	}
}
