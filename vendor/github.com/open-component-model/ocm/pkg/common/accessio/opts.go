// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessio

import (
	"io"
	"os"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common/compression"
	"github.com/open-component-model/ocm/pkg/errors"
)

type Options interface {
	Option

	SetFileFormat(FileFormat)
	GetFileFormat() *FileFormat

	SetPathFileSystem(vfs.FileSystem)
	GetPathFileSystem() vfs.FileSystem

	SetRepresentation(vfs.FileSystem)
	GetRepresentation() vfs.FileSystem

	SetFile(vfs.File)
	GetFile() vfs.File

	SetReader(closer io.ReadCloser)
	GetReader() io.ReadCloser

	ValidForPath(path string) error
	WriterFor(path string, mode vfs.FileMode) (io.WriteCloser, error)

	DefaultFormat(fmt FileFormat)
	Default()

	DefaultForPath(path string) error
}

type StandardOptions struct {
	// FilePath is the path of the repository base in the filesystem
	FileFormat *FileFormat `json:"fileFormat"`
	// FileSystem is the virtual filesystem to evaluate the file path. Default is the OS filesytem
	// or the filesystem defined as base filesystem for the context
	// This configuration option is not available for the textual representation of
	// the repository specification
	PathFileSystem vfs.FileSystem `json:"-"`
	// Representation is the virtual filesystem to represent the active repository cache.
	// This configuration option is not available for the textual representation of
	// the repository specification
	Representation vfs.FileSystem `json:"-"`
	// File is an opened file object to use instead of the path and path filesystem
	// It should never be closed if given to support temporary files
	File vfs.File `json:"-"`
	// Reader provides a one time access to the content (archive xontent only)
	// The resulting access is therefore temporarily and cannot be written back
	// to its origin, but to other destinations.
	Reader io.ReadCloser `json:"-"`
}

var _ Options = (*StandardOptions)(nil)

func (o *StandardOptions) SetFileFormat(format FileFormat) {
	o.FileFormat = &format
}

func (o *StandardOptions) GetFileFormat() *FileFormat {
	return o.FileFormat
}

func (o *StandardOptions) SetPathFileSystem(fs vfs.FileSystem) {
	o.PathFileSystem = fs
}

func (o *StandardOptions) GetPathFileSystem() vfs.FileSystem {
	return o.PathFileSystem
}

func (o *StandardOptions) SetRepresentation(fs vfs.FileSystem) {
	o.Representation = fs
}

func (o *StandardOptions) GetRepresentation() vfs.FileSystem {
	return o.Representation
}

func (o *StandardOptions) SetFile(file vfs.File) {
	o.File = file
}

func (o *StandardOptions) GetFile() vfs.File {
	return o.File
}

func (o *StandardOptions) SetReader(r io.ReadCloser) {
	o.Reader = r
}

func (o *StandardOptions) GetReader() io.ReadCloser {
	return o.Reader
}

func (o *StandardOptions) ApplyOption(options Options) error {
	if o.PathFileSystem != nil {
		options.SetPathFileSystem(o.PathFileSystem)
	}
	if o.Representation != nil {
		options.SetRepresentation(o.Representation)
	}
	if o.FileFormat != nil {
		options.SetFileFormat(*o.FileFormat)
	}
	if o.File != nil {
		options.SetFile(o.File)
	}
	if o.Reader != nil {
		options.SetReader(o.Reader)
	}
	return nil
}

var _osfs = osfs.New()

func (o *StandardOptions) Default() {
	if o.PathFileSystem == nil {
		o.PathFileSystem = _osfs
	}
}

func (o *StandardOptions) DefaultFormat(fmt FileFormat) {
	if o.FileFormat == nil {
		o.FileFormat = &fmt
	}
}

func (o *StandardOptions) DefaultForPath(path string) error {
	if err := o.ValidForPath(path); err != nil {
		return err
	}
	if o.FileFormat == nil {
		var fmt *FileFormat
		var err error
		switch {
		case o.Reader != nil:
			r, _, err := compression.AutoDecompress(o.Reader)
			if err == nil {
				o.Reader = AddCloser(r, o.Reader)
				f := FormatTar
				fmt = &f
			}
		case o.File != nil:
			fmt, err = DetectFormatForFile(o.File)
		default:
			fmt, err = DetectFormat(path, o.PathFileSystem)
		}
		if err == nil {
			o.FileFormat = fmt
		}
		return err
	}
	return nil
}

func (o *StandardOptions) ValidForPath(path string) error {
	count := 0
	if path != "" {
		count++
	}
	if o.File != nil {
		count++
	}
	if o.Reader != nil {
		count++
	}
	if count > 1 {
		return errors.ErrInvalid("only path,, file or reader can be set")
	}
	return nil
}

func (o *StandardOptions) WriterFor(path string, mode vfs.FileMode) (io.WriteCloser, error) {
	if err := o.ValidForPath(path); err != nil {
		return nil, err
	}
	var writer io.WriteCloser
	var err error
	if o.File == nil {
		writer, err = o.PathFileSystem.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode&0o666)
	} else {
		writer = NopWriteCloser(o.File)
		err = o.File.Truncate(0)
	}
	return writer, err
}

// ApplyOptions applies the given list options on these options.
func ApplyOptions(opts Options, olist ...Option) error {
	for _, opt := range olist {
		if opt != nil {
			if err := opt.ApplyOption(opts); err != nil {
				return err
			}
		}
	}
	return nil
}

// Option is the interface to specify different archive options.
type Option interface {
	ApplyOption(options Options) error
}

// PathFileSystem set the evaluation filesystem for the path name.
func PathFileSystem(fs vfs.FileSystem) Option {
	return optPfs{fs}
}

type optPfs struct {
	vfs.FileSystem
}

// ApplyOption applies the configured path filesystem.
func (o optPfs) ApplyOption(options Options) error {
	options.SetPathFileSystem(o.FileSystem)
	return nil
}

// RepresentationFileSystem set the evaltuation filesystem for the path name.
func RepresentationFileSystem(fs vfs.FileSystem) Option {
	return optRfs{fs}
}

type optRfs struct {
	vfs.FileSystem
}

// ApplyOption applies the configured path filesystem.
func (o optRfs) ApplyOption(options Options) error {
	options.SetRepresentation(o.FileSystem)
	return nil
}

// File set open file to use.
func File(file vfs.File) Option {
	return optF{file}
}

type optF struct {
	vfs.File
}

// ApplyOption applies the configured open file.
func (o optF) ApplyOption(options Options) error {
	options.SetFile(o.File)
	return nil
}

// Reader set open reader to use.
func Reader(reader io.ReadCloser) Option {
	return optR{reader}
}

type optR struct {
	io.ReadCloser
}

// ApplyOption applies the configured open file.
func (o optR) ApplyOption(options Options) error {
	options.SetReader(o.ReadCloser)
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func AccessOptions(opts Options, list ...Option) (Options, error) {
	if opts == nil {
		opts = &StandardOptions{}
	}
	err := ApplyOptions(opts, list...)
	if err != nil {
		return nil, err
	}
	opts.Default()
	return opts, nil
}
