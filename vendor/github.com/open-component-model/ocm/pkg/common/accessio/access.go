// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessio

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mandelsoft/filepath/pkg/filepath"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/errors"
)

var (
	ErrClosed   = errors.ErrClosed()
	ErrReadOnly = errors.ErrReadOnly()
)

const (
	KIND_BLOB      = "blob"
	KIND_MEDIATYPE = "media type"
)

func ErrBlobNotFound(digest digest.Digest) error {
	return errors.ErrNotFound(KIND_BLOB, digest.String())
}

func IsErrBlobNotFound(err error) bool {
	return errors.IsErrNotFoundKind(err, KIND_BLOB)
}

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

// BlobAccess describes the access to a blob.
type BlobAccess interface {
	DataAccess
	DigestSource
	MimeType

	// DigestKnown reports whether digest is already known
	DigestKnown() bool
	// Size returns the blob size
	Size() int64
}

// _blobAccess is to be used for private embedded fields.
type _blobAccess = BlobAccess

// TemporaryBlobAccess describes a blob with temporary allocated external resources.
// They will be releases, when the close method is called.
type TemporaryBlobAccess interface {
	_blobAccess
	IsValid() bool
}

// _temporaryBlobAccess is to be used for private embedded fields.
type _temporaryBlobAccess = TemporaryBlobAccess

type temporaryBlob struct {
	_blobAccess
}

// TemporaryBlobAccessForBlob returns a temporary blob for any blob, which gets
// invalidated whenever closed.
func TemporaryBlobAccessForBlob(blob BlobAccess) TemporaryBlobAccess {
	return &temporaryBlob{blob}
}

func (b *temporaryBlob) IsValid() bool {
	return b._blobAccess != nil
}

func (b *temporaryBlob) Close() error {
	if b._blobAccess == nil {
		return errors.ErrInvalid("blob access")
	}
	err := b._blobAccess.Close()
	b._blobAccess = nil
	return err
}

type DigestSource interface {
	// Digest returns the blob digest
	Digest() digest.Digest
}

////////////////////////////////////////////////////////////////////////////////

type NopCloser struct{}

type _nopCloser = NopCloser

func (NopCloser) Close() error {
	return nil
}

////////////////////////////////////////////////////////////////////////////////

type readerAccess struct {
	_nopCloser
	reader func() (io.ReadCloser, error)
	origin string
}

var _ DataSource = (*readerAccess)(nil)

func DataAccessForReaderFunction(reader func() (io.ReadCloser, error), origin string) DataAccess {
	return &readerAccess{reader: reader, origin: origin}
}

func (a *readerAccess) Get() (data []byte, err error) {
	r, err := a.Reader()
	if err != nil {
		return nil, err
	}
	defer errors.PropagateError(&err, r.Close)

	buf := bytes.Buffer{}
	_, err = io.Copy(&buf, r)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read %s", a.origin)
	}
	return buf.Bytes(), nil
}

func (a *readerAccess) Reader() (io.ReadCloser, error) {
	r, err := a.reader()
	if err != nil {
		return nil, errors.Wrapf(err, "errors getting reader for %s", a.origin)
	}
	return r, nil
}

func (a *readerAccess) Origin() string {
	return a.origin
}

////////////////////////////////////////////////////////////////////////////////

type dataAccess struct {
	_nopCloser
	fs   vfs.FileSystem
	path string
}

var _ DataSource = (*dataAccess)(nil)

func DataAccessForFile(fs vfs.FileSystem, path string) DataAccess {
	return &dataAccess{fs: fs, path: path}
}

func (a *dataAccess) Get() ([]byte, error) {
	data, err := vfs.ReadFile(a.fs, a.path)
	if err != nil {
		return nil, errors.Wrapf(err, "file %q", a.path)
	}
	return data, nil
}

func (a *dataAccess) Reader() (io.ReadCloser, error) {
	file, err := a.fs.Open(a.path)
	if err != nil {
		return nil, errors.Wrapf(err, "file %q", a.path)
	}
	return file, nil
}

func (a *dataAccess) Origin() string {
	return a.path
}

////////////////////////////////////////////////////////////////////////////////

type bytesAccess struct {
	_nopCloser
	data   []byte
	origin string
}

func DataAccessForBytes(data []byte, origin ...string) DataSource {
	path := ""
	if len(origin) > 0 {
		path = filepath.Join(origin...)
	}
	return &bytesAccess{data: data, origin: path}
}

func DataAccessForString(data string, origin ...string) DataSource {
	return DataAccessForBytes([]byte(data), origin...)
}

func (a *bytesAccess) Get() ([]byte, error) {
	return a.data, nil
}

func (a *bytesAccess) Reader() (io.ReadCloser, error) {
	return ReadCloser(bytes.NewReader(a.data)), nil
}

func (a *bytesAccess) Origin() string {
	return a.origin
}

////////////////////////////////////////////////////////////////////////////////

// AnnotatedBlobAccess provides access to the original underlying data source.
type AnnotatedBlobAccess[T DataAccess] interface {
	_blobAccess
	Source() T
}

type blobAccess[T DataAccess] struct {
	lock     sync.RWMutex
	digest   digest.Digest
	size     int64
	mimeType string
	closed   atomic.Bool
	access   T
}

const (
	BLOB_UNKNOWN_SIZE   = int64(-1)
	BLOB_UNKNOWN_DIGEST = digest.Digest("")
)

// BlobAccessForDataAccess wraps the general access object into a blob access.
// It closes the wrapped access, if closed.
func BlobAccessForDataAccess[T DataAccess](digest digest.Digest, size int64, mimeType string, access T) AnnotatedBlobAccess[T] {
	return &blobAccess[T]{
		digest:   digest,
		size:     size,
		mimeType: mimeType,
		access:   access,
	}
}

func BlobAccessForString(mimeType string, data string) BlobAccess {
	return BlobAccessForData(mimeType, []byte(data))
}

func BlobAccessForData(mimeType string, data []byte) BlobAccess {
	return &blobAccess[DataAccess]{
		digest:   digest.FromBytes(data),
		size:     int64(len(data)),
		mimeType: mimeType,
		access:   DataAccessForBytes(data),
	}
}

func (b *blobAccess[T]) Close() error {
	b.lock.Lock()
	defer b.lock.Unlock()
	if !b.closed.Load() {
		tmp := b.access
		b.closed.Store(true)
		return tmp.Close()
	}
	return ErrClosed
}

func (b *blobAccess[T]) Get() ([]byte, error) {
	if b.closed.Load() {
		return nil, ErrClosed
	}
	return b.access.Get()
}

func (b *blobAccess[T]) Reader() (io.ReadCloser, error) {
	if b.closed.Load() {
		return nil, ErrClosed
	}
	return b.access.Reader()
}

func (b *blobAccess[T]) Source() T {
	return b.access
}

func (b *blobAccess[T]) MimeType() string {
	return b.mimeType
}

func (b *blobAccess[T]) DigestKnown() bool {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.digest != ""
}

func (b *blobAccess[T]) Digest() digest.Digest {
	b.lock.Lock()
	defer b.lock.Unlock()
	if b.digest == "" {
		b.update()
	}
	return b.digest
}

func (b *blobAccess[T]) Size() int64 {
	b.lock.Lock()
	defer b.lock.Unlock()
	if b.size < 0 {
		b.update()
	}
	return b.size
}

func (b *blobAccess[T]) update() error {
	reader, err := b.Reader()
	if err != nil {
		return err
	}

	defer reader.Close()
	count := NewCountingReader(reader)

	digest, err := digest.Canonical.FromReader(count)
	if err != nil {
		return err
	}

	b.size = count.Size()
	b.digest = digest

	return nil
}

////////////////////////////////////////////////////////////////////////////////

type mimeBlob struct {
	_blobAccess
	mimetype string
}

func BlobWithMimeType(mimeType string, blob BlobAccess) BlobAccess {
	return &mimeBlob{blob, mimeType}
}

func (b *mimeBlob) MimeType() string {
	return b.mimetype
}

////////////////////////////////////////////////////////////////////////////////

type fileBlobAccess struct {
	dataAccess
	mimeType string
}

var _ BlobAccess = (*fileBlobAccess)(nil)

func BlobAccessForFile(mimeType string, path string, fss ...vfs.FileSystem) BlobAccess {
	return &fileBlobAccess{
		mimeType:   mimeType,
		dataAccess: dataAccess{fs: FileSystem(fss...), path: path},
	}
}

func (f *fileBlobAccess) Size() int64 {
	size := BLOB_UNKNOWN_SIZE
	fi, err := f.fs.Stat(f.path)
	if err == nil {
		size = fi.Size()
	}
	return size
}

func (f *fileBlobAccess) MimeType() string {
	return f.mimeType
}

func (f *fileBlobAccess) DigestKnown() bool {
	return false
}

func (f *fileBlobAccess) Digest() digest.Digest {
	r, err := f.Reader()
	if err != nil {
		return ""
	}
	defer r.Close()
	d, err := digest.FromReader(r)
	if err != nil {
		return ""
	}
	return d
}

////////////////////////////////////////////////////////////////////////////////

type blobNopCloser struct {
	_blobAccess
}

func BlobNopCloser(blob BlobAccess) BlobAccess {
	return &blobNopCloser{blob}
}

func (b *blobNopCloser) Close() error {
	return nil
}

////////////////////////////////////////////////////////////////////////////////

type MultiViewBlobAccess struct {
	refs   ReferencableCloser
	access BlobAccess
}

func NewMultiViewBlobAccess(acc BlobAccess) *MultiViewBlobAccess {
	return &MultiViewBlobAccess{
		refs:   NewRefCloser(acc, true),
		access: acc,
	}
}

func (m *MultiViewBlobAccess) View() (BlobAccess, error) {
	v, err := m.refs.View(false)
	if err != nil {
		return nil, err
	}
	return &blobAccessView{v, m.access}, nil
}

type blobAccessView struct {
	view   CloserView
	access BlobAccess
}

func (b *blobAccessView) Close() error {
	return b.view.Close()
}

func (b *blobAccessView) IsClosed() bool {
	return b.view.IsClosed()
}

func (b *blobAccessView) Get() (result []byte, err error) {
	return result, b.view.Execute(func() error {
		result, err = b.access.Get()
		if err != nil {
			return fmt.Errorf("unable to get access: %w", err)
		}

		return nil
	})
}

func (b *blobAccessView) Reader() (result io.ReadCloser, err error) {
	return result, b.view.Execute(func() error {
		result, err = b.access.Reader()
		if err != nil {
			return fmt.Errorf("unable to read access: %w", err)
		}

		return nil
	})
}

func (b *blobAccessView) Digest() (result digest.Digest) {
	err := b.view.Execute(func() error {
		result = b.access.Digest()
		return nil
	})
	if err != nil {
		return BLOB_UNKNOWN_DIGEST
	}
	return
}

func (b *blobAccessView) MimeType() string {
	return b.access.MimeType()
}

func (b *blobAccessView) DigestKnown() bool {
	return b.access.DigestKnown()
}

func (b *blobAccessView) Size() (result int64) {
	err := b.view.Execute(func() error {
		result = b.access.Size()
		return nil
	})
	if err != nil {
		return BLOB_UNKNOWN_SIZE
	}
	return
}

////////////////////////////////////////////////////////////////////////////////

type temporaryBlobAccess struct {
	_blobAccess
}

func TemporaryBlobAccessFor(blob BlobAccess) TemporaryBlobAccess {
	if t, ok := blob.(TemporaryBlobAccess); ok {
		return t
	}
	return &temporaryBlobAccess{blob}
}

func (b *temporaryBlobAccess) IsValid() bool {
	return true
}

////////////////////////////////////////////////////////////////////////////////

type TemporaryFileSystemBlobAccess interface {
	_temporaryBlobAccess
	FileSystem() vfs.FileSystem
	Path() string
}

type temporaryFileBlob struct {
	_blobAccess
	lock       sync.Mutex
	temp       vfs.File
	filesystem vfs.FileSystem
}

func TempFileBlobAccess(mime string, fs vfs.FileSystem, temp vfs.File) TemporaryFileSystemBlobAccess {
	return &temporaryFileBlob{
		_blobAccess: BlobAccessForFile(mime, temp.Name(), fs),
		filesystem:  fs,
		temp:        temp,
	}
}

func (b *temporaryFileBlob) IsValid() bool {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.temp != nil
}

func (b *temporaryFileBlob) Close() error {
	b.lock.Lock()
	defer b.lock.Unlock()
	if b.temp != nil {
		list := errors.ErrListf("temporary blob")
		list.Add(b.temp.Close())
		list.Add(b.filesystem.Remove(b.temp.Name()))
		b.temp = nil
		b._blobAccess = nil
		return list.Result()
	}
	return nil
}

func (b *temporaryFileBlob) FileSystem() vfs.FileSystem {
	return b.filesystem
}

func (b *temporaryFileBlob) Path() string {
	return b.temp.Name()
}

// TempFile holds a temporary file that should be kept open.
// Close should never be called directly.
// It can be passed to another responsibility realm by calling Release
// For example to be transformed into a TemporaryBlobAccess.
// Close will close and remove an unreleased file and does
// nothing if it has been released.
// If it has been releases the new realm is responsible.
// to close and remove it.
type TempFile struct {
	lock       sync.Mutex
	temp       vfs.File
	filesystem vfs.FileSystem
}

func NewTempFile(fs vfs.FileSystem, dir string, pattern string) (*TempFile, error) {
	temp, err := vfs.TempFile(fs, dir, pattern)
	if err != nil {
		return nil, err
	}
	return &TempFile{
		temp:       temp,
		filesystem: fs,
	}, nil
}

func (t *TempFile) Name() string {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.temp.Name()
}

func (t *TempFile) FileSystem() vfs.FileSystem {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.filesystem
}

func (t *TempFile) Release() vfs.File {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.temp != nil {
		t.temp.Sync()
	}
	tmp := t.temp
	t.temp = nil
	return tmp
}

func (t *TempFile) Writer() io.Writer {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.temp
}

func (t *TempFile) Sync() error {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.temp.Sync()
}

func (t *TempFile) AsBlob(mime string) TemporaryFileSystemBlobAccess {
	return TempFileBlobAccess(mime, t.filesystem, t.Release())
}

func (t *TempFile) Close() error {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.temp != nil {
		name := t.temp.Name()
		t.temp.Close()
		t.temp = nil
		return t.filesystem.Remove(name)
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

type referencingBlobAccess struct {
	lock   sync.Mutex
	closed bool
	closer func() error
	_blobAccess
}

func ReferencingBlobAccess(b BlobAccess, closer func() error) TemporaryBlobAccess {
	return &referencingBlobAccess{closer: closer, _blobAccess: b}
}

func (r *referencingBlobAccess) IsValid() bool {
	r.lock.Lock()
	defer r.lock.Unlock()
	return !r.closed
}

func (r *referencingBlobAccess) Close() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.closed {
		return ErrClosed
	}
	if r.closer != nil {
		if err := r.closer(); err != nil {
			return err
		}
	}
	r.closed = true
	return r._blobAccess.Close()
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
