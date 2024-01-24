// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blobaccess

import (
	"bytes"
	"compress/gzip"
	"io"
	"sync"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/blobaccess/bpi"
	compression2 "github.com/open-component-model/ocm/pkg/common/compression"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/mime"
)

////////////////////////////////////////////////////////////////////////////////

type compression struct {
	blob BlobAccess
}

var _ bpi.BlobAccessBase = (*compression)(nil)

func (c *compression) Close() error {
	return c.blob.Close()
}

func (c *compression) Get() ([]byte, error) {
	r, err := c.blob.Reader()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	rr, _, err := compression2.AutoDecompress(r)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)

	w := gzip.NewWriter(buf)
	_, err = io.Copy(w, rr)
	w.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type reader struct {
	wait sync.WaitGroup
	io.ReadCloser
	err error
}

func (r *reader) Close() error {
	err := r.ReadCloser.Close()
	r.wait.Wait()
	return errors.Join(err, r.err)
}

func (c *compression) Reader() (io.ReadCloser, error) {
	r, err := c.blob.Reader()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	rr, _, err := compression2.AutoDecompress(r)
	if err != nil {
		return nil, err
	}
	pr, pw := io.Pipe()
	cw := gzip.NewWriter(pw)

	outr := &reader{ReadCloser: pr}
	outr.wait.Add(1)

	go func() {
		_, err := io.Copy(cw, rr)
		outr.err = errors.Join(err, cw.Close(), pw.Close())
		outr.wait.Done()
	}()
	return outr, nil
}

func (c *compression) Digest() digest.Digest {
	return BLOB_UNKNOWN_DIGEST
}

func (c *compression) MimeType() string {
	m := c.blob.MimeType()
	if mime.IsGZip(m) {
		return m
	}
	return m + "+gzip"
}

func (c *compression) DigestKnown() bool {
	return false
}

func (c *compression) Size() int64 {
	return BLOB_UNKNOWN_SIZE
}

func WithCompression(blob BlobAccess) (BlobAccess, error) {
	b, err := blob.Dup()
	if err != nil {
		return nil, err
	}
	return bpi.NewBlobAccessForBase(&compression{
		blob: b,
	}), nil
}

////////////////////////////////////////////////////////////////////////////////

type decompression struct {
	blob BlobAccess
}

var _ bpi.BlobAccessBase = (*decompression)(nil)

func (c *decompression) Close() error {
	return c.blob.Close()
}

func (c *decompression) Get() ([]byte, error) {
	r, err := c.blob.Reader()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	rr, _, err := compression2.AutoDecompress(r)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, rr)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *decompression) Reader() (io.ReadCloser, error) {
	r, err := c.blob.Reader()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	rr, _, err := compression2.AutoDecompress(r)
	return rr, err
}

func (c *decompression) Digest() digest.Digest {
	return BLOB_UNKNOWN_DIGEST
}

func (c *decompression) MimeType() string {
	m := c.blob.MimeType()
	if !mime.IsGZip(m) {
		return m
	}
	return m[:len(m)-5]
}

func (c *decompression) DigestKnown() bool {
	return false
}

func (c *decompression) Size() int64 {
	return BLOB_UNKNOWN_SIZE
}

func WithDecompression(blob BlobAccess) (BlobAccess, error) {
	b, err := blob.Dup()
	if err != nil {
		return nil, err
	}
	return bpi.NewBlobAccessForBase(&decompression{
		blob: b,
	}), nil
}
