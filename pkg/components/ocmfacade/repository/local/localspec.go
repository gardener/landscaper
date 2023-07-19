package local

import (
	"github.com/gardener/landscaper/pkg/components/ocmfacade/repository"
	"github.com/mandelsoft/filepath/pkg/filepath"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	. "github.com/open-component-model/ocm/pkg/exception"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

const (
	Type   = repository.LocalType
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
	FilePath                    string         `json:"filePath"`
	PathFileSystem              vfs.FileSystem `json:"-"`
}

func NewRepositorySpecV1(filePath string, pathFileSystem ...vfs.FileSystem) (*repository.RepositorySpec, error) {
	fs := utils.Optional(pathFileSystem...)
	spec := &repository.RepositorySpec{
		InternalVersionedTypedObject: runtime.NewInternalVersionedTypedObject[cpi.RepositorySpec](versions, Type),
		CompDescFs:                   fs,
		CompDescDirPath:              filePath,
		BlobFs:                       fs,
		BlobDirPath:                  filepath.Join(filePath, "blobs"),
	}
	return spec, nil
}

type converterV1 struct{}

func (_ converterV1) ConvertFrom(in *repository.RepositorySpec) (*RepositorySpecV1, error) {
	return &RepositorySpecV1{
		ObjectVersionedType: runtime.NewVersionedObjectType(in.Type),
		PathFileSystem:      in.CompDescFs,
		FilePath:            in.CompDescDirPath,
	}, nil
}

func (_ converterV1) ConvertTo(in *RepositorySpecV1) (*repository.RepositorySpec, error) {
	return &repository.RepositorySpec{
		InternalVersionedTypedObject: runtime.NewInternalVersionedTypedObject[cpi.RepositorySpec](versions, in.Type),
		CompDescFs:                   in.PathFileSystem,
		CompDescDirPath:              in.FilePath,
		BlobFs:                       in.PathFileSystem,
		BlobDirPath:                  filepath.Join(in.FilePath, "blobs"),
	}, nil
}

///////////////////////////////////////////////////////////////////////////////

type RepositorySpecV2 struct {
	runtime.ObjectVersionedType `json:",inline"`
	CompDescFs                  vfs.FileSystem `json:"-"`
	CompDescDirPath             string         `json:"compDescDirPath"`
	BlobFs                      vfs.FileSystem `json:"-"`
	BlobDirPath                 string         `json:"blobDirPath"`
}

func NewRepositorySpecV2(compDescFs vfs.FileSystem, compDescDirPath string, blobDirPath string, blobFs ...vfs.FileSystem) (*repository.RepositorySpec, error) {
	bfs := utils.Optional(blobFs...)
	if bfs == nil {
		bfs = compDescFs
	}
	spec := &repository.RepositorySpec{
		InternalVersionedTypedObject: runtime.NewInternalVersionedTypedObject[cpi.RepositorySpec](versions, TypeV2),
		CompDescFs:                   compDescFs,
		CompDescDirPath:              compDescDirPath,
		BlobFs:                       bfs,
		BlobDirPath:                  blobDirPath,
	}
	return spec, nil
}

type converterV2 struct{}

func (_ converterV2) ConvertFrom(in *repository.RepositorySpec) (*RepositorySpecV2, error) {
	return &RepositorySpecV2{
		ObjectVersionedType: runtime.NewVersionedObjectType(in.Type),
		CompDescFs:          in.CompDescFs,
		CompDescDirPath:     in.CompDescDirPath,
		BlobFs:              in.BlobFs,
		BlobDirPath:         in.BlobDirPath,
	}, nil
}

func (_ converterV2) ConvertTo(in *RepositorySpecV2) (*repository.RepositorySpec, error) {
	return &repository.RepositorySpec{
		InternalVersionedTypedObject: runtime.NewInternalVersionedTypedObject[cpi.RepositorySpec](versions, in.Type),
		CompDescFs:                   in.CompDescFs,
		CompDescDirPath:              in.CompDescDirPath,
		BlobFs:                       in.BlobFs,
		BlobDirPath:                  in.BlobDirPath,
	}, nil
}
