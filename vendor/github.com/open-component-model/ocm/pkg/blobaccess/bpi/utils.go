// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bpi

import (
	"io"
	"sync"
	"sync/atomic"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/iotools"
)

type _dataAccess = DataAccess

type baseAccess interface {
	base() BlobAccessBase
}

func Cast[I interface{}](acc BlobAccess) I {
	var _nil I

	var b BlobAccessBase = acc

	for b != nil {
		if i, ok := b.(I); ok {
			return i
		}
		if i, ok := b.(baseAccess); ok {
			b = i.base()
		} else {
			b = nil
		}
	}
	return _nil
}

////////////////////////////////////////////////////////////////////////////////

type blobAccess struct {
	_dataAccess

	lock     sync.RWMutex
	digest   digest.Digest
	size     int64
	mimeType string
}

func (b *blobAccess) MimeType() string {
	return b.mimeType
}

func (b *blobAccess) DigestKnown() bool {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.digest != ""
}

func (b *blobAccess) Digest() digest.Digest {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b._Digest()
}

func (b *blobAccess) _Digest() digest.Digest {
	if b.digest == "" {
		b.update()
	}
	return b.digest
}

func (b *blobAccess) Size() int64 {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b._Size()
}

func (b *blobAccess) _Size() int64 {
	if b.size < 0 {
		b.update()
	}
	return b.size
}

func (b *blobAccess) update() error {
	reader, err := b.Reader()
	if err != nil {
		return err
	}

	defer reader.Close()
	count := iotools.NewCountingReader(reader)

	digest, err := digest.Canonical.FromReader(count)
	if err != nil {
		return err
	}

	b.size = count.Size()
	b.digest = digest

	return nil
}

type closableBlobAccess struct {
	blobAccess
	closed atomic.Bool
}

func (b *closableBlobAccess) Close() error {
	b.lock.Lock()
	defer b.lock.Unlock()
	if !b.closed.Load() {
		tmp := b._dataAccess
		b.closed.Store(true)
		b._dataAccess = nil
		return tmp.Close()
	}
	return ErrClosed
}

func (b *closableBlobAccess) Get() ([]byte, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.closed.Load() {
		return nil, ErrClosed
	}
	return b.blobAccess.Get()
}

func (b *closableBlobAccess) Reader() (io.ReadCloser, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.closed.Load() {
		return nil, ErrClosed
	}
	return b.blobAccess.Reader()
}

func (b *closableBlobAccess) Digest() digest.Digest {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.closed.Load() {
		return b.digest
	}
	return b.blobAccess._Digest()
}

func (b *closableBlobAccess) Size() int64 {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.closed.Load() {
		return b.size
	}
	return b.blobAccess._Size()
}

// BaseAccessForDataAccess is used for a general data.
// It calculated the metadata on-the-fly for the content
// of the data access. The content may not be changing.
func BaseAccessForDataAccess(mime string, acc DataAccess) BlobAccessBase {
	return &closableBlobAccess{
		blobAccess: blobAccess{mimeType: mime, _dataAccess: acc, digest: BLOB_UNKNOWN_DIGEST, size: BLOB_UNKNOWN_SIZE},
	}
}

func BaseAccessForDataAccessAndMeta(mime string, acc DataAccess, dig digest.Digest, size int64) BlobAccessBase {
	return &closableBlobAccess{
		blobAccess: blobAccess{mimeType: mime, _dataAccess: acc, digest: dig, size: size},
	}
}

////////////////////////////////////////////////////////////////////////////////

// StaticBlobAccess is a BlobAccess which does not
// require finalization, therefore it can be used
// as BlobAccessProvider, also.
type StaticBlobAccess interface {
	BlobAccess
	BlobAccessProvider
}

type staticBlobAccess struct {
	blobAccess
}

func (s *staticBlobAccess) Dup() (BlobAccess, error) {
	return s, nil
}

func (s *staticBlobAccess) BlobAccess() (BlobAccess, error) {
	return s, nil
}

func (s *staticBlobAccess) Close() error {
	return nil
}

// ForStaticDataAccess is used for a data access using no closer.
// They don't require a finalization and can be used
// as long as they exist. Therefore, no ref counting
// is required and they can be used as BlobAccessProvider, also.
func ForStaticDataAccess(mime string, acc DataAccess) StaticBlobAccess {
	return &staticBlobAccess{
		blobAccess: blobAccess{mimeType: mime, _dataAccess: acc, digest: BLOB_UNKNOWN_DIGEST, size: BLOB_UNKNOWN_SIZE},
	}
}

func ForStaticDataAccessAndMeta(mime string, acc DataAccess, dig digest.Digest, size int64) StaticBlobAccess {
	return &staticBlobAccess{
		blobAccess: blobAccess{mimeType: mime, _dataAccess: acc, digest: dig, size: size},
	}
}
