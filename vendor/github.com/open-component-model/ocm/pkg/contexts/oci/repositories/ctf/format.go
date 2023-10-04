// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ctf

import (
	"sort"
	"strings"
	"sync"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/ctf/format"
	"github.com/open-component-model/ocm/pkg/errors"
)

const (
	ArtifactIndexFileName = format.ArtifactIndexFileName
	BlobsDirectoryName    = format.BlobsDirectoryName
)

var accessObjectInfo = &accessobj.DefaultAccessObjectInfo{
	DescriptorFileName:       ArtifactIndexFileName,
	ObjectTypeName:           "repository",
	ElementDirectoryName:     BlobsDirectoryName,
	ElementTypeName:          "blob",
	DescriptorHandlerFactory: NewStateHandler,
}

type Object = Repository

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
	fileFormats = map[accessio.FileFormat]FormatHandler{}
	lock        sync.RWMutex
)

func RegisterFormat(f accessobj.FormatHandler) FormatHandler {
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
	return fileFormats[name]
}

func SupportedFormats() []accessio.FileFormat {
	lock.RLock()
	defer lock.RUnlock()
	result := make([]accessio.FileFormat, 0, len(fileFormats))
	for f := range fileFormats {
		result = append(result, f)
	}
	sort.Slice(result, func(i, j int) bool { return strings.Compare(string(result[i]), string(result[j])) < 0 })
	return result
}

////////////////////////////////////////////////////////////////////////////////

func OpenFromBlob(ctx cpi.ContextProvider, acc accessobj.AccessMode, blob accessio.BlobAccess, opts ...accessio.Option) (*Object, error) {
	o, err := accessio.AccessOptions(nil, opts...)
	if err != nil {
		return nil, err
	}
	if o.GetFile() != nil || o.GetReader() != nil {
		return nil, errors.ErrInvalid("file or reader option nor possible for blob access")
	}
	reader, err := blob.Reader()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	o.SetReader(reader)
	fmt := accessio.FormatTar
	mime := blob.MimeType()
	if strings.HasSuffix(mime, "+gzip") {
		fmt = accessio.FormatTGZ
	}
	o.SetFileFormat(fmt)
	return Open(ctx, acc&accessobj.ACC_READONLY, "", 0, o)
}

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
		return h.Create(cpi.FromProvider(ctx), path, o, mode)
	}
	return h.Open(cpi.FromProvider(ctx), acc, path, o)
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
	return h.Create(ctx.OCIContext(), path, o, mode)
}

func (h *formatHandler) Open(ctx cpi.ContextProvider, acc accessobj.AccessMode, path string, opts accessio.Options) (*Object, error) {
	obj, err := h.FormatHandler.Open(accessObjectInfo, acc, path, opts)
	if err != nil {
		return nil, err
	}
	spec, err := NewRepositorySpec(acc, path, opts)
	return _Wrap(ctx, spec, obj, err)
}

func (h *formatHandler) Create(ctx cpi.ContextProvider, path string, opts accessio.Options, mode vfs.FileMode) (*Object, error) {
	obj, err := h.FormatHandler.Create(accessObjectInfo, path, opts, mode)
	if err != nil {
		return nil, err
	}
	spec, err := NewRepositorySpec(accessobj.ACC_CREATE, path, opts)
	return _Wrap(ctx, spec, obj, err)
}

// WriteToFilesystem writes the current object to a filesystem.
func (h *formatHandler) Write(obj *Object, path string, opts accessio.Options, mode vfs.FileMode) error {
	return h.FormatHandler.Write(obj.impl.base.Access(), path, opts, mode)
}
