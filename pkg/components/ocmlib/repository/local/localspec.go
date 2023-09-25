// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"github.com/mandelsoft/filepath/pkg/filepath"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	. "github.com/open-component-model/ocm/pkg/exception"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"

	"github.com/gardener/landscaper/pkg/components/ocmlib/repository"
)

const (
	Type   = repository.LocalType
	TypeV1 = Type + runtime.VersionSeparator + "v1"
)

var versions = cpi.NewRepositoryTypeVersionScheme(Type)

func init() {
	Must(versions.Register(cpi.NewRepositoryTypeByConverter[*repository.RepositorySpec, *RepositorySpecV1](Type, &converterV1{}, nil)))
	Must(versions.Register(cpi.NewRepositoryTypeByConverter[*repository.RepositorySpec, *RepositorySpecV1](TypeV1, &converterV1{}, nil)))
	cpi.RegisterRepositoryTypeVersions(versions)
}

type RepositorySpecV1 struct {
	runtime.ObjectVersionedType `json:",inline"`
	FilePath                    string         `json:"filePath"`
	PathFileSystem              vfs.FileSystem `json:"-"`
}

func NewRepositorySpecV1(filePath string, pathFileSystem ...vfs.FileSystem) (*repository.RepositorySpec, error) {
	fs := utils.Optional(pathFileSystem...)
	spec := &repository.RepositorySpec{
		InternalVersionedTypedObject: runtime.NewInternalVersionedTypedObject[cpi.RepositorySpec](versions, Type),
		FileSystem:                   fs,
		CompDescDirPath:              filePath,
		BlobDirPath:                  filepath.Join(filePath, "blobs"),
	}
	return spec, nil
}

type converterV1 struct{}

func (_ converterV1) ConvertFrom(in *repository.RepositorySpec) (*RepositorySpecV1, error) {
	return &RepositorySpecV1{
		ObjectVersionedType: runtime.NewVersionedObjectType(in.Type),
		PathFileSystem:      in.FileSystem,
		FilePath:            in.CompDescDirPath,
	}, nil
}

func (_ converterV1) ConvertTo(in *RepositorySpecV1) (*repository.RepositorySpec, error) {
	return &repository.RepositorySpec{
		InternalVersionedTypedObject: runtime.NewInternalVersionedTypedObject[cpi.RepositorySpec](versions, in.Type),
		FileSystem:                   in.PathFileSystem,
		CompDescDirPath:              in.FilePath,
		BlobDirPath:                  filepath.Join(in.FilePath, "blobs"),
	}, nil
}
