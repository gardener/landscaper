// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"io"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/opencontainers/go-digest"
)

const (
	KIND_BLOB      = "blob"
	KIND_MEDIATYPE = "media type"

	BLOB_UNKNOWN_SIZE   = int64(-1)
	BLOB_UNKNOWN_DIGEST = digest.Digest("")
)

type DataGetter interface {
	// Get returns the content as byte array
	Get() ([]byte, error)
}

type DataReader interface {
	// Reader returns a reader to incrementally access byte stream content
	Reader() (io.ReadCloser, error)
}

////////////////////////////////////////////////////////////////////////////////

// DataSource describes some data plus its origin.
type DataSource interface {
	DataAccess
	Origin() string
}

////////////////////////////////////////////////////////////////////////////////

// DataAccess describes the access to sequence of bytes.
type DataAccess interface {
	DataGetter
	DataReader
	io.Closer
}

type MimeType interface {
	// MimeType returns the mime type of the blob
	MimeType() string
}

type DigestSource interface {
	// Digest returns the blob digest
	Digest() digest.Digest
}

// BlobAccessBase describes the access to a blob.
type BlobAccessBase interface {
	DataAccess
	DigestSource
	MimeType

	// DigestKnown reports whether digest is already known
	DigestKnown() bool
	// Size returns the blob size
	Size() int64
}

// BlobAccess describes the access to a blob.
type BlobAccess interface {
	BlobAccessBase

	// Dup provides a new independently closable view.
	Dup() (BlobAccess, error)
}

type FileLocation interface {
	FileSystem() vfs.FileSystem
	Path() string
}

type BlobAccessProvider interface {
	BlobAccess() (BlobAccess, error)
}
