package inline

import (
	"encoding/json"
	"github.com/gardener/landscaper/pkg/components/ocmfacade/repository"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/mandelsoft/vfs/pkg/yamlfs"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	. "github.com/open-component-model/ocm/pkg/exception"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	Type   = repository.InlineType
	TypeV1 = Type + runtime.VersionSeparator + "v1"
	TypeV2 = Type + runtime.VersionSeparator + "v2"
)

var versions = cpi.NewRepositoryTypeVersionScheme(Type)

func init() {
	Must(versions.Register(cpi.NewRepositoryTypeByConverter[*repository.RepositorySpec, *RepositorySpecV1](Type, &converterV1{}, nil)))
	Must(versions.Register(cpi.NewRepositoryTypeByConverter[*repository.RepositorySpec, *RepositorySpecV1](TypeV1, &converterV1{}, nil)))
	Must(versions.Register(cpi.NewRepositoryTypeByConverter[*repository.RepositorySpec, *RepositorySpecV2](TypeV2, &converterV2{}, nil)))
	cpi.RegisterRepositoryTypeVersions(versions)
}

type RepositorySpecV1 struct {
	runtime.ObjectVersionedType `json:",inline"`
	CompDescFs                  vfs.FileSystem `json:"-"`
	BlobFs                      vfs.FileSystem `json:"-"`
}

func NewRepositorySpecV1(compDescFs vfs.FileSystem, blobFs vfs.FileSystem) (*repository.RepositorySpec, error) {
	spec := &repository.RepositorySpec{
		InternalVersionedTypedObject: runtime.NewInternalVersionedTypedObject[cpi.RepositorySpec](versions, Type),
		CompDescFs:                   nil,
		CompDescDirPath:              "",
		BlobFs:                       nil,
		BlobDirPath:                  "",
	}
	return spec, nil
}

type converterV1 struct{}

func (_ converterV1) ConvertFrom(in *repository.RepositorySpec) (*RepositorySpecV1, error) {
	return &RepositorySpecV1{
		ObjectVersionedType: runtime.NewVersionedObjectType(in.Type),
		CompDescFs:          in.CompDescFs,
		BlobFs:              in.BlobFs,
	}, nil
}

func (_ converterV1) ConvertTo(in *RepositorySpecV1) (*repository.RepositorySpec, error) {
	return &repository.RepositorySpec{
		InternalVersionedTypedObject: runtime.NewInternalVersionedTypedObject[cpi.RepositorySpec](versions, in.Type),
		CompDescFs:                   in.CompDescFs,
		CompDescDirPath:              "",
		BlobFs:                       in.BlobFs,
		BlobDirPath:                  "",
	}, nil
}

///////////////////////////////////////////////////////////////////////////////

type RepositorySpecV2 struct {
	runtime.ObjectVersionedType `json:",inline"`
	CompDescFs                  json.RawMessage `json:"compDescFs"`
	CompDescDirPath             string          `json:"compDescDirPath"`
	BlobFs                      json.RawMessage `json:"blobFs"`
	BlobDirPath                 string          `json:"blobDirPath"`
}

func NewRepositorySpecV2(compDescFs vfs.FileSystem, compDescDirPath string, blobFs vfs.FileSystem, blobDirPath string) (*repository.RepositorySpec, error) {
	spec := &repository.RepositorySpec{
		InternalVersionedTypedObject: runtime.NewInternalVersionedTypedObject[cpi.RepositorySpec](versions, TypeV2),
		CompDescFs:                   compDescFs,
		CompDescDirPath:              compDescDirPath,
		BlobFs:                       blobFs,
		BlobDirPath:                  blobDirPath,
	}
	return spec, nil
}

type converterV2 struct{}

func (_ converterV2) ConvertFrom(in *repository.RepositorySpec) (*RepositorySpecV2, error) {
	cfs, err := yamlfs.New([]byte{})
	if err != nil {
		return nil, err
	}
	err = vfs.CopyDir(in.CompDescFs, "", cfs, "")
	if err != nil {
		return nil, err
	}

	compdescYaml, err := cfs.Data()
	if err != nil {
		return nil, err
	}

	bfs, err := yamlfs.New([]byte{})
	if err != nil {
		return nil, err
	}
	err = vfs.CopyDir(in.CompDescFs, "", bfs, "")
	if err != nil {
		return nil, err
	}

	blobYaml, err := cfs.Data()
	if err != nil {
		return nil, err
	}

	return &RepositorySpecV2{
		ObjectVersionedType: runtime.NewVersionedObjectType(in.Type),
		CompDescFs:          compdescYaml,
		CompDescDirPath:     in.CompDescDirPath,
		BlobFs:              blobYaml,
		BlobDirPath:         in.BlobDirPath,
	}, nil
}

func (_ converterV2) ConvertTo(in *RepositorySpecV2) (*repository.RepositorySpec, error) {
	var cmemfs vfs.FileSystem
	var bmemfs vfs.FileSystem

	if in.CompDescFs != nil {
		cfs, err := yamlfs.New(in.CompDescFs)
		if err != nil {
			return nil, err
		}
		cmemfs = memoryfs.New()
		err = vfs.CopyDir(cfs, "", cmemfs, "")
		if err != nil {
			return nil, err
		}
	}

	if in.BlobFs != nil {
		bfs, err := yamlfs.New(in.BlobFs)
		if err != nil {
			return nil, err
		}
		bmemfs = memoryfs.New()
		err = vfs.CopyDir(bfs, "", bmemfs, "")
		if err != nil {
			return nil, err
		}
	}

	return &repository.RepositorySpec{
		InternalVersionedTypedObject: runtime.NewInternalVersionedTypedObject[cpi.RepositorySpec](versions, in.Type),
		CompDescFs:                   cmemfs,
		CompDescDirPath:              in.CompDescDirPath,
		BlobFs:                       bmemfs,
		BlobDirPath:                  in.BlobDirPath,
	}, nil
}
