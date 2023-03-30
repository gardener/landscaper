// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessobj

import (
	"fmt"
	"io"
	"os"

	"github.com/mandelsoft/filepath/pkg/filepath"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/errors"
)

var FormatDirectory = DirectoryHandler{}

func init() {
	RegisterFormat(FormatDirectory)
}

type DirectoryHandler struct{}

// ApplyOption applies the configured path filesystem.
func (o DirectoryHandler) ApplyOption(options accessio.Options) error {
	options.SetFileFormat(o.Format())
	return nil
}

func (_ DirectoryHandler) Format() accessio.FileFormat {
	return accessio.FormatDirectory
}

func (_ DirectoryHandler) Open(info AccessObjectInfo, acc AccessMode, path string, opts accessio.Options) (*AccessObject, error) {
	if err := opts.ValidForPath(path); err != nil {
		return nil, err
	}
	if opts.GetFile() != nil || opts.GetReader() != nil {
		return nil, errors.ErrNotSupported("file or reader option")
	}
	fs, err := projectionfs.New(opts.GetPathFileSystem(), path)
	if err != nil {
		return nil, fmt.Errorf("unable to create projected filesystem from path %s: %w", path, err)
	}
	opts.SetRepresentation(fs) // TODO: use of temporary copy
	return NewAccessObject(info, acc, fs, nil, nil, os.ModePerm)
}

func (_ DirectoryHandler) Create(info AccessObjectInfo, path string, opts accessio.Options, mode vfs.FileMode) (*AccessObject, error) {
	if err := opts.ValidForPath(path); err != nil {
		return nil, err
	}
	if opts.GetFile() != nil || opts.GetReader() != nil {
		return nil, errors.ErrNotSupported("file or reader option")
	}
	err := opts.GetPathFileSystem().Mkdir(path, mode)
	if err != nil {
		return nil, err
	}
	rep, err := projectionfs.New(opts.GetPathFileSystem(), path)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create projected filesystem from path %s", path)
	}
	opts.SetRepresentation(rep)
	return NewAccessObject(info, ACC_CREATE, rep, nil, nil, mode)
}

// WriteToFilesystem writes the current object to a filesystem.
func (_ DirectoryHandler) Write(obj *AccessObject, path string, opts accessio.Options, mode vfs.FileMode) error {
	// create the directory structure with the content directory
	if err := opts.GetPathFileSystem().MkdirAll(filepath.Join(path, obj.info.GetElementDirectoryName()), mode|0o400); err != nil {
		return errors.Wrapf(err, "unable to create output directory %q", path)
	}

	_, err := obj.updateDescriptor()
	if err != nil {
		return errors.Wrapf(err, "unable to update descriptor")
	}

	// copy descriptor
	err = vfs.CopyFile(obj.fs, obj.info.GetDescriptorFileName(), opts.GetPathFileSystem(), filepath.Join(path, obj.info.GetDescriptorFileName()))
	if err != nil {
		return errors.Wrapf(err, "unable to copy file '%s'", obj.info.GetDescriptorFileName())
	}

	// Copy additional files
	for _, f := range obj.info.GetAdditionalFiles(obj.fs) {
		ok, err := vfs.IsFile(obj.fs, f)
		if err != nil {
			return errors.Wrapf(err, "cannot check for file %q", f)
		}
		if ok {
			err = vfs.CopyFile(obj.fs, f, opts.GetPathFileSystem(), filepath.Join(path, f))
			if err != nil {
				return errors.Wrapf(err, "unable to copy file '%s'", f)
			}
		}
	}

	// copy all content
	fileInfos, err := vfs.ReadDir(obj.fs, obj.info.GetElementDirectoryName())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrapf(err, "unable to read '%s'", obj.info.GetElementDirectoryName())
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			continue
		}
		inpath := obj.info.SubPath(fileInfo.Name())
		outpath := filepath.Join(path, inpath)
		content, err := obj.fs.Open(inpath)
		if err != nil {
			return errors.Wrapf(err, "unable to open input %s %q", obj.info.GetElementTypeName(), inpath)
		}
		out, err := opts.GetPathFileSystem().OpenFile(outpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode|0o666)
		if err != nil {
			return errors.Wrapf(err, "unable to open output %s %q", obj.info.GetElementTypeName(), outpath)
		}
		if _, err := io.Copy(out, content); err != nil {
			return errors.Wrapf(err, "unable to copy %s from %q to %q", obj.info.GetElementTypeName(), inpath, outpath)
		}
		if err := out.Close(); err != nil {
			return errors.Wrapf(err, "unable to close output %s %s", obj.info.GetElementTypeName(), outpath)
		}
		if err := content.Close(); err != nil {
			return errors.Wrapf(err, "unable to close input %s %s", obj.info.GetElementTypeName(), outpath)
		}
	}

	return nil
}
