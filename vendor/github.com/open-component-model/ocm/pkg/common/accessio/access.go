// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessio

import (
	"io"
	"math/rand"
	"time"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/iotools"
	"github.com/open-component-model/ocm/pkg/refmgmt"
)

var (
	ErrClosed   = refmgmt.ErrClosed
	ErrReadOnly = errors.ErrReadOnly()
)

const (
	// Deprecated: use blobaccess.KIND_BLOB.
	KIND_BLOB = blobaccess.KIND_BLOB
	// Deprecated: use blobaccess.KIND_MEDIATYPE.
	KIND_MEDIATYPE = blobaccess.KIND_MEDIATYPE
)

// Deprecated: use blobaccess.ErrBlobNotFound.
func ErrBlobNotFound(digest digest.Digest) error {
	return errors.ErrNotFound(blobaccess.KIND_BLOB, digest.String())
}

// Deprecated: use blobaccess.IsErrBlobNotFound.
func IsErrBlobNotFound(err error) bool {
	return errors.IsErrNotFoundKind(err, blobaccess.KIND_BLOB)
}

////////////////////////////////////////////////////////////////////////////////

// DataSource describes some data plus its origin.
// Deprecated: use blobaccess.DataSource.
type DataSource = blobaccess.DataSource

////////////////////////////////////////////////////////////////////////////////

// DataAccess describes the access to sequence of bytes.
// Deprecated: use blobaccess.DataAccess.
type DataAccess = blobaccess.DataAccess

// BlobAccess describes the access to a blob.
// Deprecated: use blobaccess.BlobAccess.
type BlobAccess = blobaccess.BlobAccess

////////////////////////////////////////////////////////////////////////////////

type NopCloser = iotools.NopCloser

////////////////////////////////////////////////////////////////////////////////

// Deprecated: use blobaccess.DataAccessForReaderFunction.
func DataAccessForReaderFunction(reader func() (io.ReadCloser, error), origin string) blobaccess.DataAccess {
	return blobaccess.DataAccessForReaderFunction(reader, origin)
}

////////////////////////////////////////////////////////////////////////////////

// Deprecated: use blobaccess.DataAccessForFile.
func DataAccessForFile(fs vfs.FileSystem, path string) blobaccess.DataAccess {
	return blobaccess.DataAccessForFile(fs, path)
}

////////////////////////////////////////////////////////////////////////////////

// Deprecated: use blobaccess.DataAccessForBytes.
func DataAccessForBytes(data []byte, origin ...string) blobaccess.DataSource {
	return blobaccess.DataAccessForBytes(data, origin...)
}

// Deprecated: use blobaccess.DataAccessForString.
func DataAccessForString(data string, origin ...string) blobaccess.DataSource {
	return blobaccess.DataAccessForBytes([]byte(data), origin...)
}

////////////////////////////////////////////////////////////////////////////////

// AnnotatedBlobAccess provides access to the original underlying data source.
// Deprecated: use blobaccess.AnnotatedBlobAccess.
type AnnotatedBlobAccess[T blobaccess.DataAccess] interface {
	blobaccess.BlobAccess
	Source() T
}

// BlobAccessForDataAccess wraps the general access object into a blob access.
// It closes the wrapped access, if closed.
// Deprecated: use blobaccess.ForDataAccess.
func BlobAccessForDataAccess[T blobaccess.DataAccess](digest digest.Digest, size int64, mimeType string, access T) blobaccess.AnnotatedBlobAccess[T] {
	return blobaccess.ForDataAccess[T](digest, size, mimeType, access)
}

// Deprecated: use blobaccess.ForString.
func BlobAccessForString(mimeType string, data string) blobaccess.BlobAccess {
	return blobaccess.ForData(mimeType, []byte(data))
}

// Deprecated: use blobaccess.ForData.
func BlobAccessForData(mimeType string, data []byte) blobaccess.BlobAccess {
	return blobaccess.ForData(mimeType, data)
}

////////////////////////////////////////////////////////////////////////////////

// BlobWithMimeType changes the mime type for a blob access
// by wrapping the given blob access. It does NOT provide
// a new view for the given blob access, so closing the resulting
// blob access will directly close the backing blob access.
// Deprecated: use blobaccess.WithMimeType.
func BlobWithMimeType(mimeType string, blob blobaccess.BlobAccess) blobaccess.BlobAccess {
	return blobaccess.WithMimeType(mimeType, blob)
}

////////////////////////////////////////////////////////////////////////////////

// Deprecated: use blobaccess.ForFile.
func BlobAccessForFile(mimeType string, path string, fss ...vfs.FileSystem) blobaccess.BlobAccess {
	return blobaccess.ForFile(mimeType, path, fss...)
}

// Deprecated: use blobaccess.ForFileWithCloser.
func BlobAccessForFileWithCloser(closer io.Closer, mimeType string, path string, fss ...vfs.FileSystem) blobaccess.BlobAccess {
	return blobaccess.ForFileWithCloser(closer, mimeType, path, fss...)
}

////////////////////////////////////////////////////////////////////////////////

// Deprecated: use blobaccess.ForTemporaryFile.
func BlobAccessForTemporaryFile(mime string, temp vfs.File, fss ...vfs.FileSystem) blobaccess.BlobAccess {
	return blobaccess.ForTemporaryFile(mime, temp, fss...)
}

// Deprecated: use blobaccess.ForTemporaryFilePath.
func BlobAccessForTemporaryFilePath(mime string, temp string, fss ...vfs.FileSystem) blobaccess.BlobAccess {
	return blobaccess.ForTemporaryFilePath(mime, temp, fss...)
}

////////////////////////////////////////////////////////////////////////////////

// Deprecated: use blobaccess.NewTempFile.
func NewTempFile(fs vfs.FileSystem, dir string, pattern string) (*blobaccess.TempFile, error) {
	return blobaccess.NewTempFile(dir, pattern, fs)
}

////////////////////////////////////////////////////////////////////////////////

type retriableError struct {
	wrapped error
}

func IsRetriableError(err error) bool {
	if err == nil {
		return false
	}
	return errors.IsA(err, &retriableError{})
}

func RetriableError(err error) error {
	if err == nil {
		return nil
	}
	return &retriableError{err}
}

func RetriableError1[T any](r T, err error) (T, error) {
	if err == nil {
		return r, nil
	}
	return r, &retriableError{err}
}

func RetriableError2[S, T any](s S, r T, err error) (S, T, error) {
	if err == nil {
		return s, r, nil
	}
	return s, r, &retriableError{err}
}

func (e *retriableError) Error() string {
	return e.wrapped.Error()
}

func (e *retriableError) Unwrap() error {
	return e.wrapped
}

func Retry(cnt int, d time.Duration, f func() error) error {
	for {
		err := f()
		if err == nil || cnt <= 0 || !IsRetriableError(err) {
			return err
		}
		jitter := time.Duration(rand.Int63n(int64(d))) //nolint: gosec // just an random number
		d = 2*d + (d/2-jitter)/10
		cnt--
	}
}

func Retry1[T any](cnt int, d time.Duration, f func() (T, error)) (T, error) {
	for {
		r, err := f()
		if err == nil || cnt <= 0 || !IsRetriableError(err) {
			return r, err
		}
		jitter := time.Duration(rand.Int63n(int64(d))) //nolint: gosec // just an random number
		d = 2*d + (d/2-jitter)/10
		cnt--
	}
}

func Retry2[S, T any](cnt int, d time.Duration, f func() (S, T, error)) (S, T, error) {
	for {
		s, t, err := f()
		if err == nil || cnt <= 0 || !IsRetriableError(err) {
			return s, t, err
		}
		jitter := time.Duration(rand.Int63n(int64(d))) //nolint: gosec // just an random number
		d = 2*d + (d/2-jitter)/10
		cnt--
	}
}
