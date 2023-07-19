package repository

import (
	"errors"
	"github.com/gardener/landscaper/pkg/components/ocmfacade/repository/attrs/blobvfs"
	"github.com/gardener/landscaper/pkg/components/ocmfacade/repository/attrs/compvfs"
	"github.com/gardener/landscaper/pkg/components/ocmfacade/repository/attrs/localrootfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/runtime"
)

type RepositorySpec struct {
	runtime.InternalVersionedTypedObject[cpi.RepositorySpec]
	CompDescFs      vfs.FileSystem
	CompDescDirPath string
	BlobFs          vfs.FileSystem
	BlobDirPath     string
}

func (r RepositorySpec) MarshalJSON() ([]byte, error) {
	return runtime.MarshalVersionedTypedObject(&r)
	// return cpi.MarshalConvertedAccessSpec(cpi.DefaultContext(), &a)
}

func (r *RepositorySpec) Repository(ctx cpi.Context, creds credentials.Credentials) (cpi.Repository, error) {
	// Set default file systems if not passed to spec
	switch r.GetKind() {
	case LocalType:
		if r.CompDescFs == nil {
			r.CompDescFs = localrootfs.Get(ctx)
			if r.CompDescFs == nil {
				return nil, errors.New("no localrootfs attribute set, check whether local registry configuration is given")
			}
		}

		if r.BlobFs == nil {
			r.BlobFs = localrootfs.Get(ctx)
			if r.BlobFs == nil {
				return nil, errors.New("no localrootfs attribute set, check whether local registry configuration is given")
			}
		}
	case InlineType:
		if r.CompDescFs == nil {
			r.CompDescFs = compvfs.Get(ctx)
			if r.CompDescFs == nil {
				return nil, errors.New("no localrootfs attribute set, check whether local registry configuration is given")
			}
		}

		if r.BlobFs == nil {
			r.BlobFs = blobvfs.Get(ctx)
			if r.BlobFs == nil {
				return nil, errors.New("no localrootfs attribute set, check whether local registry configuration is given")
			}
		}
	}

	return NewRepositoryFromSpec(ctx, r)
}

func (r *RepositorySpec) AsUniformSpec(ctx cpi.Context) *cpi.UniformRepositorySpec {
	return nil
}
