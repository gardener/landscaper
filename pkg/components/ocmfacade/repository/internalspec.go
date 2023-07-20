package repository

import (
	"github.com/gardener/landscaper/pkg/components/ocmfacade/repository/attrs/localrootfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/virtual"
	"github.com/open-component-model/ocm/pkg/runtime"
)

type ComponentDescriptorProvider interface {
	List() ([]*compdesc.ComponentDescriptor, error)
}

type RepositorySpec struct {
	runtime.InternalVersionedTypedObject[cpi.RepositorySpec]
	CompDescFs      vfs.FileSystem // Component Descriptor Provider Interface (Liste oder Filesystem)
	CompDescDirPath string
	BlobFs          vfs.FileSystem
	BlobDirPath     string
}

func (r RepositorySpec) MarshalJSON() ([]byte, error) {
	return runtime.MarshalVersionedTypedObject(&r)
	// return cpi.MarshalConvertedAccessSpec(cpi.DefaultContext(), &a)
}

// Case 1: alles aus repository context (alles yaml filesystem)
// condition:
// 		kind = inline
// 		compdescfs != nil
// handling:
// 		blobfs = blobvfs attr
//
// creation because of:
//
// -----> Installation.ComponentReference

// Case 2: descriptor aus installation
// kind = inline type = inline
// --> installation.inlinecd != nil -> compvfs = inlinecd
// --> installation.inlinebp != nil -> blobvfs = inlinebp

// Case 3: descriptor ref local:
// kind = local
// -> CompDescFs = nil -> CompDescFs = localrootfs
// -> BlobFs = nil -> BlobFs = localrootfs

func (r *RepositorySpec) Repository(ctx cpi.Context, creds credentials.Credentials) (cpi.Repository, error) {
	var err error

	descriptorfs := r.CompDescFs
	if r.GetKind() == LocalType && descriptorfs == nil {
		descriptorfs = localrootfs.Get(ctx)
	}

	prov := NewFilesystemCompDescProvider(r.CompDescDirPath, descriptorfs)
	blobfs := r.BlobFs
	if r.GetKind() == LocalType && blobfs == nil {
		blobfs = localrootfs.Get(ctx)
	}

	if r.BlobDirPath != "" {
		blobfs, err = projectionfs.New(blobfs, r.BlobDirPath)
		if err != nil {
			return nil, err
		}
	} else {
	}
	acc, err := NewAccess(prov, blobfs)
	if err != nil {
		return nil, err
	}
	return virtual.NewRepository(ctx, acc), nil
}

func (r *RepositorySpec) AsUniformSpec(ctx cpi.Context) *cpi.UniformRepositorySpec {
	return nil
}
