package blobaccess

import (
	"io"
	"sync"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/blobaccess/bpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/finalizer"
	"github.com/open-component-model/ocm/pkg/mime"
	"github.com/open-component-model/ocm/pkg/refmgmt"
	"github.com/open-component-model/ocm/pkg/utils"
)

var ErrClosed = refmgmt.ErrClosed

func ErrBlobNotFound(digest digest.Digest) error {
	return errors.ErrNotFound(KIND_BLOB, digest.String())
}

func IsErrBlobNotFound(err error) bool {
	return errors.IsErrNotFoundKind(err, KIND_BLOB)
}

////////////////////////////////////////////////////////////////////////////////

// Validatable is an optional interface for DataAccess
// implementations or any other object, which might reach
// an error state. The error can then be queried with
// the method ErrorProvider.Validate.
// This is used to support objects with access methods not
// returning an error. If the object is not valid,
// those methods return an unknown/default state, but
// the object should be queryable for its state.
type Validatable = utils.Validatable

// Validate checks whether a blob access
// is in error state. If yes, an appropriate
// error is returned.
func Validate(o BlobAccess) error {
	return utils.ValidateObject(o)
}

////////////////////////////////////////////////////////////////////////////////

type blobprovider struct {
	blob BlobAccess
}

var _ BlobAccessProvider = (*blobprovider)(nil)

func (b *blobprovider) BlobAccess() (BlobAccess, error) {
	return b.blob.Dup()
}

func (b *blobprovider) Close() error {
	return b.blob.Close()
}

// ProviderForBlobAccess provides subsequent bloc accesses
// as long as the given blob access is not closed.
// If required the blob can be closed with the additionally
// provided Close method.
// ATTENTION: the underlying BlobAccess wil not be closed
// as long as the provider is not closed, but the BlobProvider
// interface is no io.Closer.
// To be on the safe side, this method should only be called
// with static blob access, featuring a NOP closer without
// anny attached external resources, which should be released.
func ProviderForBlobAccess(blob BlobAccess) *blobprovider {
	return &blobprovider{blob}
}

////////////////////////////////////////////////////////////////////////////////

// ForString wraps a string into a BlobAccess, which does not need a close.
func ForString(media string, data string) BlobAccess {
	if media == "" {
		media = mime.MIME_TEXT
	}
	return ForData(media, []byte(data))
}

func ProviderForString(mime, data string) BlobAccessProvider {
	return bpi.BlobAccessProviderFunction(func() (bpi.BlobAccess, error) {
		return ForString(mime, data), nil
	})
}

// ForData wraps data into a BlobAccess, which does not need a close.
func ForData(media string, data []byte) BlobAccess {
	if media == "" {
		media = mime.MIME_OCTET
	}
	return bpi.ForStaticDataAccessAndMeta(media, DataAccessForBytes(data), digest.FromBytes(data), int64(len(data)))
}

func ProviderForData(mime string, data []byte) BlobAccessProvider {
	return bpi.BlobAccessProviderFunction(func() (bpi.BlobAccess, error) {
		return ForData(mime, data), nil
	})
}

type _blobAccess = BlobAccess

////////////////////////////////////////////////////////////////////////////////

type fileBlobAccess struct {
	fileDataAccess
	mimeType string
}

var (
	_ BlobAccess   = (*fileBlobAccess)(nil)
	_ FileLocation = (*fileBlobAccess)(nil)
)

func (f *fileBlobAccess) FileSystem() vfs.FileSystem {
	return f.fs
}

func (f *fileBlobAccess) Path() string {
	return f.path
}

func (f *fileBlobAccess) Dup() (BlobAccess, error) {
	return f, nil
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

// ForFile wraps a file path into a BlobAccess, which does not need a close.
func ForFile(mime string, path string, fss ...vfs.FileSystem) BlobAccess {
	return &fileBlobAccess{
		mimeType:       mime,
		fileDataAccess: fileDataAccess{fs: utils.FileSystem(fss...), path: path},
	}
}

func ProviderForFile(mime string, path string, fss ...vfs.FileSystem) BlobAccessProvider {
	return bpi.BlobAccessProviderFunction(func() (bpi.BlobAccess, error) {
		return ForFile(mime, path, fss...), nil
	})
}

type fileBlobAccessView struct {
	_blobAccess
	access *fileDataAccess
}

var (
	_ BlobAccess   = (*fileBlobAccessView)(nil)
	_ FileLocation = (*fileBlobAccessView)(nil)
)

func (f *fileBlobAccessView) Dup() (BlobAccess, error) {
	b, err := f._blobAccess.Dup()
	if err != nil {
		return nil, err
	}
	return &fileBlobAccessView{b, f.access}, nil
}

func (f *fileBlobAccessView) FileSystem() vfs.FileSystem {
	return f.access.fs
}

func (f *fileBlobAccessView) Path() string {
	return f.access.path
}

func ForFileWithCloser(closer io.Closer, mime string, path string, fss ...vfs.FileSystem) BlobAccess {
	fb := &fileBlobAccess{fileDataAccess{fs: utils.FileSystem(fss...), path: path}, mime}
	return &fileBlobAccessView{
		bpi.NewBlobAccessForBase(fb, closer),
		&fb.fileDataAccess,
	}
}

////////////////////////////////////////////////////////////////////////////////

// AnnotatedBlobAccess provides access to the original underlying data source.
type AnnotatedBlobAccess[T DataAccess] interface {
	_blobAccess
	Source() T
}

type annotatedBlobAccessView[T DataAccess] struct {
	_blobAccess
	id         finalizer.ObjectIdentity
	annotation T
}

func (a *annotatedBlobAccessView[T]) Close() error {
	return a._blobAccess.Close()
}

func (a *annotatedBlobAccessView[T]) Dup() (BlobAccess, error) {
	b, err := a._blobAccess.Dup()
	if err != nil {
		return nil, err
	}
	return &annotatedBlobAccessView[T]{
		id:          finalizer.NewObjectIdentity(a.id.String()),
		_blobAccess: b,
		annotation:  a.annotation,
	}, nil
}

func (a *annotatedBlobAccessView[T]) Source() T {
	return a.annotation
}

// ForDataAccess wraps the general access object into a blob access.
// It closes the wrapped access, if closed.
// If the wrapped data access does not need a close, the BlobAccess
// does not need a close, also.
func ForDataAccess[T DataAccess](digest digest.Digest, size int64, mimeType string, access T) AnnotatedBlobAccess[T] {
	a := bpi.BaseAccessForDataAccessAndMeta(mimeType, access, digest, size)

	return &annotatedBlobAccessView[T]{
		id:          finalizer.NewObjectIdentity("annotatedBlobAccess"),
		_blobAccess: bpi.NewBlobAccessForBase(a),
		annotation:  access,
	}
}

////////////////////////////////////////////////////////////////////////////////

type temporaryFileBlob struct {
	_blobAccess
	lock       sync.Mutex
	path       string
	file       vfs.File
	filesystem vfs.FileSystem
}

var (
	_ bpi.BlobAccessBase = (*temporaryFileBlob)(nil)
	_ FileLocation       = (*temporaryFileBlob)(nil)
)

func (b *temporaryFileBlob) Validate() error {
	b.lock.Lock()
	defer b.lock.Unlock()
	if b.path == "" {
		return ErrClosed
	}
	ok, err := vfs.Exists(b.filesystem, b.path)
	if err != nil {
		return err
	}
	if !ok {
		return errors.ErrNotFound("file", b.path)
	}
	return nil
}

func (b *temporaryFileBlob) Close() error {
	b.lock.Lock()
	defer b.lock.Unlock()
	if b.path != "" {
		list := errors.ErrListf("temporary blob")
		if b.file != nil {
			list.Add(b.file.Close())
		}
		list.Add(b.filesystem.Remove(b.path))
		b.path = ""
		b.file = nil
		b._blobAccess = nil
		return list.Result()
	}
	return nil
}

func (b *temporaryFileBlob) FileSystem() vfs.FileSystem {
	return b.filesystem
}

func (b *temporaryFileBlob) Path() string {
	return b.path
}

func ForTemporaryFile(mime string, temp vfs.File, fss ...vfs.FileSystem) BlobAccess {
	return bpi.NewBlobAccessForBase(&temporaryFileBlob{
		_blobAccess: ForFile(mime, temp.Name(), fss...),
		filesystem:  utils.FileSystem(fss...),
		path:        temp.Name(),
		file:        temp,
	})
}

func ForTemporaryFilePath(mime string, temp string, fss ...vfs.FileSystem) BlobAccess {
	return bpi.NewBlobAccessForBase(&temporaryFileBlob{
		_blobAccess: ForFile(mime, temp, fss...),
		filesystem:  utils.FileSystem(fss...),
		path:        temp,
	})
}

////////////////////////////////////////////////////////////////////////////////

type mimeBlob struct {
	_blobAccess
	mimetype string
}

// WithMimeType changes the mime type for a blob access
// by wrapping the given blob access. It does NOT provide
// a new view for the given blob access, so closing the resulting
// blob access will directly close the backing blob access.
func WithMimeType(mimeType string, blob BlobAccess) BlobAccess {
	return &mimeBlob{blob, mimeType}
}

func (b *mimeBlob) Dup() (BlobAccess, error) {
	n, err := b._blobAccess.Dup()
	if err != nil {
		return nil, err
	}
	return &mimeBlob{n, b.mimetype}, nil
}

func (b *mimeBlob) MimeType() string {
	return b.mimetype
}

////////////////////////////////////////////////////////////////////////////////

type blobNopCloser struct {
	_blobAccess
}

func NonClosable(blob BlobAccess) BlobAccess {
	return &blobNopCloser{blob}
}

func (b *blobNopCloser) Close() error {
	return nil
}
