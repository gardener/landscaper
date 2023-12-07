// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessobj

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/mime"
	"github.com/open-component-model/ocm/pkg/utils/tarutils"
)

type FileSystemBlobAccess struct {
	sync.RWMutex
	base *AccessObject
}

func NewFileSystemBlobAccess(access *AccessObject) *FileSystemBlobAccess {
	return &FileSystemBlobAccess{
		base: access,
	}
}

func (a *FileSystemBlobAccess) Access() *AccessObject {
	return a.base
}

func (a *FileSystemBlobAccess) IsReadOnly() bool {
	return a.base.IsReadOnly()
}

func (a *FileSystemBlobAccess) IsClosed() bool {
	return a.base.IsClosed()
}

func (a *FileSystemBlobAccess) Write(path string, mode vfs.FileMode, opts ...accessio.Option) error {
	return a.base.Write(path, mode, opts...)
}

func (a *FileSystemBlobAccess) Update() error {
	return a.base.Update()
}

func (a *FileSystemBlobAccess) Close() error {
	return a.base.Close()
}

func (a *FileSystemBlobAccess) GetState() State {
	return a.base.GetState()
}

// DigestPath returns the path to the blob for a given name.
func (a *FileSystemBlobAccess) DigestPath(digest digest.Digest) string {
	return a.BlobPath(common.DigestToFileName(digest))
}

// BlobPath returns the path to the blob for a given name.
func (a *FileSystemBlobAccess) BlobPath(name string) string {
	return a.base.GetInfo().SubPath(name)
}

func (a *FileSystemBlobAccess) GetBlobData(digest digest.Digest) (int64, blobaccess.DataAccess, error) {
	if a.IsClosed() {
		return blobaccess.BLOB_UNKNOWN_SIZE, nil, accessio.ErrClosed
	}
	path := a.DigestPath(digest)
	if ok, err := vfs.FileExists(a.base.GetFileSystem(), path); ok {
		return blobaccess.BLOB_UNKNOWN_SIZE, blobaccess.DataAccessForFile(a.base.GetFileSystem(), path), nil
	} else {
		if err != nil {
			return blobaccess.BLOB_UNKNOWN_SIZE, nil, err
		}
		return blobaccess.BLOB_UNKNOWN_SIZE, nil, blobaccess.ErrBlobNotFound(digest)
	}
}

func (a *FileSystemBlobAccess) GetBlobDataByName(name string) (blobaccess.DataAccess, error) {
	if a.IsClosed() {
		return nil, accessio.ErrClosed
	}

	path := a.BlobPath(name)
	if ok, err := vfs.IsDir(a.base.GetFileSystem(), path); ok {
		tempfile, err := blobaccess.NewTempFile(os.TempDir(), "COMPARCH")
		if err != nil {
			return nil, err
		}
		err = tarutils.PackFsIntoTar(a.base.GetFileSystem(), path, tempfile.Writer(), tarutils.TarFileSystemOptions{})
		if err != nil {
			return nil, err
		}
		return tempfile.AsBlob(mime.MIME_TAR), nil
	} else {
		if err != nil {
			return nil, err
		}

		if ok, err := vfs.FileExists(a.base.GetFileSystem(), path); ok {
			return blobaccess.DataAccessForFile(a.base.GetFileSystem(), path), nil
		} else {
			if err != nil {
				return nil, err
			}
			return nil, blobaccess.ErrBlobNotFound(digest.Digest(name))
		}
	}
}

func (a *FileSystemBlobAccess) AddBlob(blob blobaccess.BlobAccess) error {
	if a.base.IsClosed() {
		return accessio.ErrClosed
	}

	if a.base.IsReadOnly() {
		return accessio.ErrReadOnly
	}

	path := a.DigestPath(blob.Digest())

	if ok, err := vfs.FileExists(a.base.GetFileSystem(), path); ok {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check if '%s' file exists: %w", path, err)
	}

	r, err := blob.Reader()
	if err != nil {
		return fmt.Errorf("unable to read blob: %w", err)
	}

	defer r.Close()
	w, err := a.base.GetFileSystem().OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, a.base.GetMode()&0o666)
	if err != nil {
		return fmt.Errorf("unable to open file '%s': %w", path, err)
	}

	_, err = io.Copy(w, r)
	if err != nil {
		w.Close()

		return fmt.Errorf("unable to copy blob content: %w", err)
	}
	return w.Close()
}
