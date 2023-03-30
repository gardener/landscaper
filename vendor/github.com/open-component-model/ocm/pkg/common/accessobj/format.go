// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessobj

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/errors"
)

const KIND_FILEFORMAT = accessio.KIND_FILEFORMAT

const (
	DirMode  = 0o755
	FileMode = 0o644
)

var ModTime = time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)

type FileFormat = accessio.FileFormat

type FormatHandler interface {
	accessio.Option

	Format() accessio.FileFormat

	Open(info AccessObjectInfo, acc AccessMode, path string, opts accessio.Options) (*AccessObject, error)
	Create(info AccessObjectInfo, path string, opts accessio.Options, mode vfs.FileMode) (*AccessObject, error)
	Write(obj *AccessObject, path string, opts accessio.Options, mode vfs.FileMode) error
}

////////////////////////////////////////////////////////////////////////////////

var (
	fileFormats = map[FileFormat]FormatHandler{}
	lock        sync.RWMutex
)

func RegisterFormat(f FormatHandler) {
	lock.Lock()
	defer lock.Unlock()
	fileFormats[f.Format()] = f
}

func GetFormat(name FileFormat) FormatHandler {
	lock.RLock()
	defer lock.RUnlock()
	return fileFormats[name]
}

func GetFormats() map[accessio.FileFormat]FormatHandler {
	lock.RLock()
	defer lock.RUnlock()

	m := map[accessio.FileFormat]FormatHandler{}
	for k, v := range fileFormats {
		m[k] = v
	}
	return m
}

////////////////////////////////////////////////////////////////////////////////

type Closer interface {
	Close(*AccessObject) error
}

type CloserFunction func(*AccessObject) error

func (f CloserFunction) Close(obj *AccessObject) error {
	return f(obj)
}

////////////////////////////////////////////////////////////////////////////////

type Setup interface {
	Setup(vfs.FileSystem) error
}

type SetupFunction func(vfs.FileSystem) error

func (f SetupFunction) Setup(fs vfs.FileSystem) error {
	return f(fs)
}

////////////////////////////////////////////////////////////////////////////////

type fsCloser struct {
	closer Closer
}

func FSCloser(closer Closer) Closer {
	return &fsCloser{closer}
}

func (f fsCloser) Close(obj *AccessObject) error {
	err := errors.ErrListf("cannot close %s", obj.info.GetObjectTypeName())
	if f.closer != nil {
		err.Add(f.closer.Close(obj))
	}
	err.Add(vfs.Cleanup(obj.fs))
	return err.Result()
}

type StandardReaderHandler interface {
	Write(obj *AccessObject, path string, opts accessio.Options, mode vfs.FileMode) error
	NewFromReader(info AccessObjectInfo, acc AccessMode, in io.Reader, opts accessio.Options, closer Closer) (*AccessObject, error)
}

func DefaultOpenOptsFileHandling(kind string, info AccessObjectInfo, acc AccessMode, path string, opts accessio.Options, handler StandardReaderHandler) (*AccessObject, error) {
	if err := opts.ValidForPath(path); err != nil {
		return nil, err
	}
	var file vfs.File
	var err error
	var closer Closer

	reader := opts.GetReader()
	switch {
	case reader != nil:
		defer reader.Close()
	case opts.GetFile() == nil:
		// we expect that the path point to a tar
		file, err = opts.GetPathFileSystem().Open(path)
		if err != nil {
			return nil, fmt.Errorf("unable to open %s from %s: %w", kind, path, err)
		}
		defer file.Close()
	default:
		file = opts.GetFile()
	}
	if file != nil {
		reader = file
		fi, err := file.Stat()
		if err != nil {
			return nil, err
		}
		closer = CloserFunction(func(obj *AccessObject) error { return handler.Write(obj, path, opts, fi.Mode()) })
	}
	return handler.NewFromReader(info, acc, reader, opts, closer)
}

func DefaultCreateOptsFileHandling(kind string, info AccessObjectInfo, path string, opts accessio.Options, mode vfs.FileMode, handler StandardReaderHandler) (*AccessObject, error) {
	if err := opts.ValidForPath(path); err != nil {
		return nil, err
	}
	if opts.GetReader() != nil {
		return nil, errors.ErrNotSupported("reader option not supported")
	}
	if opts.GetFile() == nil {
		ok, err := vfs.Exists(opts.GetPathFileSystem(), path)
		if err != nil {
			return nil, err
		}
		if ok {
			return nil, vfs.ErrExist
		}
	}

	return NewAccessObject(info, ACC_CREATE, opts.GetRepresentation(), nil, CloserFunction(func(obj *AccessObject) error { return handler.Write(obj, path, opts, mode) }), DirMode)
}
