// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessobj

import (
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/errors"
)

type FilesystemSetup func(fs vfs.FileSystem, mode vfs.FileMode) error

// InternalRepresentationFilesystem defaults a filesystem to temp filesystem and adapts.
func InternalRepresentationFilesystem(acc AccessMode, fs vfs.FileSystem, setup FilesystemSetup, mode vfs.FileMode) (bool, vfs.FileSystem, error) {
	var err error

	tmp := false
	if fs == nil {
		fs, err = osfs.NewTempFileSystem()
		if err != nil {
			return false, nil, err
		}
		tmp = true
	}
	if !acc.IsReadonly() && setup != nil {
		err = setup(fs, mode)
		if err != nil {
			return false, nil, err
		}
	}
	return tmp, fs, err
}

func HandleAccessMode(acc AccessMode, path string, opts accessio.Options, olist ...accessio.Option) (accessio.Options, bool, error) {
	ok := true
	o, err := accessio.AccessOptions(opts, olist...)
	if err != nil {
		return nil, false, err
	}
	if o.GetFile() == nil && o.GetReader() == nil {
		ok, err = vfs.Exists(o.GetPathFileSystem(), path)
		if err != nil {
			return o, false, err
		}
	}
	if !ok {
		if !acc.IsCreate() {
			return o, false, errors.ErrNotFoundWrap(vfs.ErrNotExist, "file", path)
		}
		if o.GetFileFormat() == nil {
			o.SetFileFormat(accessio.FormatDirectory)
		}
		return o, true, nil
	}

	err = o.DefaultForPath(path)
	return o, false, err
}
