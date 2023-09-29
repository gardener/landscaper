// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package inline

import (
	"encoding/json"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/mandelsoft/vfs/pkg/yamlfs"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
	. "github.com/open-component-model/ocm/pkg/exception"
	"github.com/open-component-model/ocm/pkg/runtime"

	"github.com/gardener/landscaper/pkg/components/ocmlib/repository"
)

const (
	Type   = repository.InlineType
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
	FileSystem                  json.RawMessage `json:"fileSystem,omitempty"`
	CompDescDirPath             string          `json:"compDescDirPath,omitempty"`
	BlobFs                      json.RawMessage `json:"blobFs,omitempty"`
	BlobFsMode                  string          `json:"blobFsMode"`
	BlobDirPath                 string          `json:"blobDirPath,omitempty"`
}

func NewRepositorySpecV1(fileSystem vfs.FileSystem, compDescDirPath string, blobFs vfs.FileSystem, blobDirPath string) (*repository.RepositorySpec, error) {
	spec := &repository.RepositorySpec{
		InternalVersionedTypedObject: runtime.NewInternalVersionedTypedObject[cpi.RepositorySpec](versions, Type),
		FileSystem:                   fileSystem,
		CompDescDirPath:              compDescDirPath,
		BlobFs:                       blobFs,
		BlobDirPath:                  blobDirPath,
	}
	return spec, nil
}

type converterV1 struct{}

func (_ converterV1) ConvertFrom(in *repository.RepositorySpec) (*RepositorySpecV1, error) {
	var err error
	var compdescYaml json.RawMessage
	var blobYaml json.RawMessage

	if fs, ok := in.FileSystem.(yamlfs.YamlFileSystem); ok {
		compdescYaml, err = fs.Data()
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("cannot serialize non-yaml filesystem")
	}

	if in.BlobFs != nil && in.BlobFs != in.FileSystem {
		if in.BlobFsMode != repository.CONTEXT {
			if fs, ok := in.BlobFs.(yamlfs.YamlFileSystem); ok {
				blobYaml, err = fs.Data()
				if err != nil {
					return nil, err
				}
			} else {
				return nil, errors.New("cannot serialize non-yaml filesystem")
			}
		}
	}

	return &RepositorySpecV1{
		ObjectVersionedType: runtime.NewVersionedObjectType(in.Type),
		FileSystem:          compdescYaml,
		CompDescDirPath:     in.CompDescDirPath,
		BlobFs:              blobYaml,
		BlobFsMode:          in.BlobFsMode,
		BlobDirPath:         in.BlobDirPath,
	}, nil
}

func (_ converterV1) ConvertTo(in *RepositorySpecV1) (*repository.RepositorySpec, error) {
	var err error
	var compfs vfs.FileSystem
	var blobfs vfs.FileSystem

	if in.FileSystem != nil {
		compfs, err = yamlfs.New(in.FileSystem)
		if err != nil {
			return nil, err
		}
	}

	if in.BlobFs != nil {
		blobfs, err = yamlfs.New(in.BlobFs)
		if err != nil {
			return nil, err
		}
	}

	return &repository.RepositorySpec{
		InternalVersionedTypedObject: runtime.NewInternalVersionedTypedObject[cpi.RepositorySpec](versions, in.Type),
		FileSystem:                   compfs,
		CompDescDirPath:              in.CompDescDirPath,
		BlobFs:                       blobfs,
		BlobFsMode:                   in.BlobFsMode,
		BlobDirPath:                  in.BlobDirPath,
	}, nil
}
