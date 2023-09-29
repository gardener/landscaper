// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package comparch

import (
	"sync"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
)

// ComponentDescriptorFileName is the name of the component-descriptor file.
const ComponentDescriptorFileName = compdesc.ComponentDescriptorFileName

// BlobsDirectoryName is the name of the blob directory in the tar.
const BlobsDirectoryName = "blobs"

var accessObjectInfo = &accessobj.DefaultAccessObjectInfo{
	DescriptorFileName:       ComponentDescriptorFileName,
	ObjectTypeName:           "artifactset",
	ElementDirectoryName:     BlobsDirectoryName,
	ElementTypeName:          "blob",
	DescriptorHandlerFactory: NewStateHandler,
}

type Object = ComponentArchive

type FormatHandler interface {
	accessio.Option

	Format() accessio.FileFormat

	Open(ctx cpi.ContextProvider, acc accessobj.AccessMode, path string, opts accessio.Options) (*Object, error)
	Create(ctx cpi.ContextProvider, path string, opts accessio.Options, mode vfs.FileMode) (*Object, error)
	Write(obj *Object, path string, opts accessio.Options, mode vfs.FileMode) error
}

type formatHandler struct {
	accessobj.FormatHandler
}

var (
	FormatDirectory = RegisterFormat(accessobj.FormatDirectory)
	FormatTAR       = RegisterFormat(accessobj.FormatTAR)
	FormatTGZ       = RegisterFormat(accessobj.FormatTGZ)
)

////////////////////////////////////////////////////////////////////////////////

var (
	fileFormats = map[accessio.FileFormat]*formatHandler{}
	lock        sync.RWMutex
)

func RegisterFormat(f accessobj.FormatHandler) *formatHandler {
	lock.Lock()
	defer lock.Unlock()
	h := &formatHandler{f}
	fileFormats[f.Format()] = h
	return h
}

func GetFormats() []string {
	lock.RLock()
	defer lock.RUnlock()
	return accessio.GetFormatsFor(fileFormats)
}

func GetFormat(name accessio.FileFormat) FormatHandler {
	lock.RLock()
	defer lock.RUnlock()
	h, ok := fileFormats[name]
	if ok {
		return h
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func Open(ctx cpi.ContextProvider, acc accessobj.AccessMode, path string, mode vfs.FileMode, opts ...accessio.Option) (*Object, error) {
	o, create, err := accessobj.HandleAccessMode(acc, path, nil, opts...)
	if err != nil {
		return nil, err
	}
	h, ok := fileFormats[*o.GetFileFormat()]
	if !ok {
		return nil, errors.ErrUnknown(accessobj.KIND_FILEFORMAT, o.GetFileFormat().String())
	}
	if create {
		return h.Create(ctx, path, o, mode)
	}
	return h.Open(ctx, acc, path, o)
}

func Create(ctx cpi.ContextProvider, acc accessobj.AccessMode, path string, mode vfs.FileMode, opts ...accessio.Option) (*Object, error) {
	o, err := accessio.AccessOptions(nil, opts...)
	if err != nil {
		return nil, err
	}
	o.DefaultFormat(accessio.FormatDirectory)
	h, ok := fileFormats[*o.GetFileFormat()]
	if !ok {
		return nil, errors.ErrUnknown(accessobj.KIND_FILEFORMAT, o.GetFileFormat().String())
	}
	return h.Create(ctx, path, o, mode)
}

////////////////////////////////////////////////////////////////////////////////

func (h *formatHandler) Open(ctx cpi.ContextProvider, acc accessobj.AccessMode, path string, opts accessio.Options) (*Object, error) {
	obj, err := h.FormatHandler.Open(accessObjectInfo, acc, path, opts)
	if err != nil {
		return nil, err
	}
	spec, err := NewRepositorySpec(acc, path, opts)
	return _Wrap(ctx, obj, spec, err)
}

func (h *formatHandler) Create(ctx cpi.ContextProvider, path string, opts accessio.Options, mode vfs.FileMode) (*Object, error) {
	obj, err := h.FormatHandler.Create(accessObjectInfo, path, opts, mode)
	if err != nil {
		return nil, err
	}
	spec, err := NewRepositorySpec(accessobj.ACC_CREATE, path, opts)
	return _Wrap(ctx, obj, spec, err)
}

// WriteToFilesystem writes the current object to a filesystem.
func (h *formatHandler) Write(obj *Object, path string, opts accessio.Options, mode vfs.FileMode) error {
	return h.FormatHandler.Write(obj.container.base.Access(), path, opts, mode)
}
