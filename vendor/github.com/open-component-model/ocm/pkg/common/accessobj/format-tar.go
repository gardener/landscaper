// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessobj

import (
	"archive/tar"
	"fmt"
	"io"
	"os"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/compression"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils/tarutils"
)

var FormatTAR = NewTarHandler()

func init() {
	RegisterFormat(FormatTAR)
}

type TarHandler struct {
	format      FileFormat
	compression compression.Algorithm
}

var _ StandardReaderHandler = (*TarHandler)(nil)

var _ FormatHandler = (*TarHandler)(nil)

func NewTarHandler() *TarHandler {
	return NewTarHandlerWithCompression(accessio.FormatTar, nil)
}

func NewTarHandlerWithCompression(format FileFormat, algorithm compression.Algorithm) *TarHandler {
	return &TarHandler{
		format:      format,
		compression: algorithm,
	}
}

// ApplyOption applies the configured path filesystem.
func (h *TarHandler) ApplyOption(options accessio.Options) error {
	options.SetFileFormat(h.Format())
	return nil
}

func (h *TarHandler) Format() accessio.FileFormat {
	return h.format
}

func (h *TarHandler) Open(info AccessObjectInfo, acc AccessMode, path string, opts accessio.Options) (*AccessObject, error) {
	return DefaultOpenOptsFileHandling(fmt.Sprintf("%s archive", h.format), info, acc, path, opts, h)
}

func (h *TarHandler) Create(info AccessObjectInfo, path string, opts accessio.Options, mode vfs.FileMode) (*AccessObject, error) {
	return DefaultCreateOptsFileHandling(fmt.Sprintf("%s archive", h.format), info, path, opts, mode, h)
}

// Write tars the current descriptor and its artifacts.
func (h *TarHandler) Write(obj *AccessObject, path string, opts accessio.Options, mode vfs.FileMode) error {
	writer, err := opts.WriterFor(path, mode)
	if err != nil {
		return fmt.Errorf("unable to write: %w", err)
	}

	defer writer.Close()

	return h.WriteToStream(obj, writer, opts)
}

func (h TarHandler) WriteToStream(obj *AccessObject, writer io.Writer, opts accessio.Options) error {
	if h.compression != nil {
		w, err := h.compression.Compressor(writer, nil, nil)
		if err != nil {
			return fmt.Errorf("unable to compress writer: %w", err)
		}
		defer w.Close()

		writer = w
	}

	// write descriptor
	err := obj.Update()
	if err != nil {
		return fmt.Errorf("unable to update access object: %w", err)
	}

	data, err := obj.state.GetBlob()
	if err != nil {
		return fmt.Errorf("unable to write to get state blob: %w", err)
	}

	tw := tar.NewWriter(writer)
	cdHeader := &tar.Header{
		Name:    obj.info.GetDescriptorFileName(),
		Size:    data.Size(),
		Mode:    FileMode,
		ModTime: ModTime,
	}

	if err := tw.WriteHeader(cdHeader); err != nil {
		return fmt.Errorf("unable to write descriptor header: %w", err)
	}

	r, err := data.Reader()
	if err != nil {
		return fmt.Errorf("unable to get reader: %w", err)
	}
	defer r.Close()

	if _, err := io.Copy(tw, r); err != nil {
		return fmt.Errorf("unable to write descriptor content: %w", err)
	}

	// Copy additional files
	for _, f := range obj.info.GetAdditionalFiles(obj.fs) {
		ok, err := vfs.IsFile(obj.fs, f)
		if err != nil {
			return errors.Wrapf(err, "cannot check for file %q", f)
		}
		if ok {
			fi, err := obj.fs.Stat(f)
			if err != nil {
				return errors.Wrapf(err, "cannot stat file %q", f)
			}
			header := &tar.Header{
				Name:    f,
				Size:    fi.Size(),
				Mode:    FileMode,
				ModTime: ModTime,
			}
			if err := tw.WriteHeader(header); err != nil {
				return errors.Wrapf(err, "unable to write descriptor header")
			}

			r, err := obj.fs.Open(f)
			if err != nil {
				return errors.Wrapf(err, "unable to get reader")
			}
			if _, err := io.Copy(tw, r); err != nil {
				r.Close()
				return errors.Wrapf(err, "unable to write file %s", f)
			}
			r.Close()
		}
	}

	// add all element content
	err = tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir,
		Name:     obj.info.GetElementDirectoryName(),
		Mode:     DirMode,
		ModTime:  ModTime,
	})
	if err != nil {
		return fmt.Errorf("unable to write %s directory: %w", obj.info.GetElementTypeName(), err)
	}

	fileInfos, err := vfs.ReadDir(obj.fs, obj.info.GetElementDirectoryName())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("unable to read %s directory: %w", obj.info.GetElementTypeName(), err)
	}

	for _, fileInfo := range fileInfos {
		path := obj.info.SubPath(fileInfo.Name())
		header := &tar.Header{
			Name:    path,
			Size:    fileInfo.Size(),
			Mode:    FileMode,
			ModTime: ModTime,
		}
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("unable to write %s header: %w", obj.info.GetElementTypeName(), err)
		}

		content, err := obj.fs.Open(path)
		if err != nil {
			return fmt.Errorf("unable to open %s: %w", obj.info.GetElementTypeName(), err)
		}
		if _, err := io.Copy(tw, content); err != nil {
			return fmt.Errorf("unable to write %s content: %w", obj.info.GetElementTypeName(), err)
		}
		if err := content.Close(); err != nil {
			return fmt.Errorf("unable to close %s %s: %w", obj.info.GetElementTypeName(), path, err)
		}
	}

	return tw.Close()
}

func (h *TarHandler) NewFromReader(info AccessObjectInfo, acc AccessMode, in io.Reader, opts accessio.Options, closer Closer) (*AccessObject, error) {
	if h.compression != nil {
		reader, err := h.compression.Decompressor(in)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		in = reader
	}
	setup := func(fs vfs.FileSystem) error {
		if err := tarutils.ExtractTarToFs(fs, in); err != nil {
			return fmt.Errorf("unable to extract tar: %w", err)
		}
		return nil
	}
	return NewAccessObject(info, acc, opts.GetRepresentation(), SetupFunction(setup), closer, DirMode)
}
