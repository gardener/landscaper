// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package repository

import (
	"encoding/json"
	"fmt"

	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/attrs/vfsattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/virtual"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	FILESYSTEM = "filesystem"
	CONTEXT    = "context"
)

type ComponentDescriptorProvider interface {
	List() ([]*compdesc.ComponentDescriptor, error)
}

type RepositorySpec struct {
	runtime.InternalVersionedTypedObject[cpi.RepositorySpec]
	FileSystem      vfs.FileSystem
	CompDescDirPath string
	BlobFs          vfs.FileSystem
	BlobFsMode      string
	BlobDirPath     string
}

func (r RepositorySpec) MarshalJSON() ([]byte, error) {
	return runtime.MarshalVersionedTypedObject(&r)
}

func (r *RepositorySpec) Key() (string, error) {
	var fs string
	var blobfs string

	if r.FileSystem != nil {
		fs = fmt.Sprintf("%p", r.FileSystem)
	} else {
		fs = "nil"
	}

	if r.BlobFs != nil {
		blobfs = fmt.Sprintf("%p", r.FileSystem)
	} else {
		blobfs = "nil"
	}

	data, err := json.Marshal(&struct {
		Type            string `json:"type"`
		FileSystem      string `json:"fileSystem"`
		CompDescDirPath string `json:"compDescDirPath"`
		BlobFs          string `json:"blobFs"`
		BlobFsMode      string `json:"blobFsMode"`
		BlobDirPath     string `json:"blobDirPath"`
	}{
		Type:            r.GetType(),
		FileSystem:      fs,
		CompDescDirPath: r.CompDescDirPath,
		BlobFs:          blobfs,
		BlobFsMode:      r.BlobFsMode,
		BlobDirPath:     r.BlobDirPath,
	})

	return string(data), err
}

func NewRepository(ctx cpi.Context, provider ComponentDescriptorProvider, blobfs vfs.FileSystem) (cpi.Repository, error) {
	acc, err := NewAccess(provider, blobfs)
	if err != nil {
		return nil, err
	}
	return virtual.NewRepository(ctx, acc), nil
}

func (r *RepositorySpec) Repository(ctx cpi.Context, creds credentials.Credentials) (cpi.Repository, error) {
	descriptorfs := r.FileSystem
	if descriptorfs == nil {
		descriptorfs = vfsattr.Get(ctx)
	}

	prov := NewFilesystemCompDescProvider(r.CompDescDirPath, descriptorfs)
	blobfs := r.BlobFs
	if blobfs == nil {
		switch r.BlobFsMode {
		case "":
			r.BlobFsMode = FILESYSTEM
			fallthrough
		case FILESYSTEM:
			blobfs = descriptorfs
		case CONTEXT:
			blobfs = vfsattr.Get(ctx)
		default:
			return nil, errors.ErrInvalid("blobFsMode", r.BlobFsMode)
		}
	}

	if r.BlobDirPath != "" {
		exists, err := vfs.DirExists(blobfs, r.BlobDirPath)
		if err != nil {
			return nil, err
		}
		if exists {
			blobfs, err = projectionfs.New(blobfs, r.BlobDirPath)
			if err != nil {
				return nil, err
			}
		}
	}
	return NewRepository(ctx, prov, blobfs)
}

func (r *RepositorySpec) AsUniformSpec(ctx cpi.Context) *cpi.UniformRepositorySpec {
	return nil
}
