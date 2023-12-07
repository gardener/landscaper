// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blobaccess

import (
	"bytes"
	"io"

	"github.com/mandelsoft/filepath/pkg/filepath"
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/iotools"
)

// DataSource describes some data plus its origin.
type DataSource interface {
	DataAccess
	Origin() string
}

////////////////////////////////////////////////////////////////////////////////

type _nopCloser = iotools.NopCloser

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

type fileDataAccess struct {
	_nopCloser
	fs   vfs.FileSystem
	path string
}

var (
	_ DataSource  = (*fileDataAccess)(nil)
	_ Validatable = (*fileDataAccess)(nil)
)

func DataAccessForFile(fs vfs.FileSystem, path string) DataAccess {
	return &fileDataAccess{fs: fs, path: path}
}

func (a *fileDataAccess) Get() ([]byte, error) {
	data, err := vfs.ReadFile(a.fs, a.path)
	if err != nil {
		return nil, errors.Wrapf(err, "file %q", a.path)
	}
	return data, nil
}

func (a *fileDataAccess) Reader() (io.ReadCloser, error) {
	file, err := a.fs.Open(a.path)
	if err != nil {
		return nil, errors.Wrapf(err, "file %q", a.path)
	}
	return file, nil
}

func (a *fileDataAccess) Validate() error {
	ok, err := vfs.Exists(a.fs, a.path)
	if err != nil {
		return err
	}
	if !ok {
		return errors.ErrNotFound("file", a.path)
	}
	return nil
}

func (a *fileDataAccess) Origin() string {
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
	return iotools.ReadCloser(bytes.NewReader(a.data)), nil
}

func (a *bytesAccess) Origin() string {
	return a.origin
}
