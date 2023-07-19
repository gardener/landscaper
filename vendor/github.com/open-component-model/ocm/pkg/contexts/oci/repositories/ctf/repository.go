// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ctf

import (
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/attrs/vfsattr"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/artifactset"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/ctf/index"
)

/*
   A common transport archive is just a folder with artifact archives.
   in tar format and an index.json file. The name of the archive
   is the digest of the artifact descriptor.

   The artifact archive is a filesystem structure with a file
   artifact-descriptor.json and a folder blobs containing
   the flat blob files with the name according to the blob digest.

   Digests used as filename will replace the ":" by a "."
*/

type Repository struct {
	cpi.Repository
	impl *RepositoryImpl
}

func (r *Repository) Write(path string, mode vfs.FileMode, opts ...accessio.Option) error {
	if r.IsClosed() {
		return cpi.ErrClosed
	}
	return r.impl.Write(path, mode, opts...)
}

func (r *Repository) Close() error { // why ???
	return r.Repository.Close()
}

////////////////////////////////////////////////////////////////////////////////

// RepositoryImpl is closed, if all views are released.
type RepositoryImpl struct {
	cpi.RepositoryImplBase

	spec *RepositorySpec
	base *artifactset.FileSystemBlobAccess
}

var _ cpi.RepositoryImpl = (*RepositoryImpl)(nil)

// New returns a new representation based repository.
func New(ctx cpi.Context, spec *RepositorySpec, setup accessobj.Setup, closer accessobj.Closer, mode vfs.FileMode) (*Repository, error) {
	if spec.GetPathFileSystem() == nil {
		spec.SetPathFileSystem(vfsattr.Get(ctx))
	}
	base, err := accessobj.NewAccessObject(accessObjectInfo, spec.AccessMode, spec.GetRepresentation(), setup, closer, mode)
	return _Wrap(ctx, spec, base, err)
}

func _Wrap(ctx cpi.ContextProvider, spec *RepositorySpec, obj *accessobj.AccessObject, err error) (*Repository, error) {
	if err != nil {
		return nil, err
	}
	impl := &RepositoryImpl{
		RepositoryImplBase: cpi.NewRepositoryImplBase(cpi.FromProvider(ctx)),
		spec:               spec,
		base:               artifactset.NewFileSystemBlobAccess(obj),
	}
	r := cpi.NewRepository(impl, "OCI CTF")
	return &Repository{r, impl}, nil
}

func (r *RepositoryImpl) GetSpecification() cpi.RepositorySpec {
	return r.spec
}

func (r *RepositoryImpl) NamespaceLister() cpi.NamespaceLister {
	return r
}

func (r *RepositoryImpl) NumNamespaces(prefix string) (int, error) {
	return len(cpi.FilterByNamespacePrefix(prefix, r.getIndex().RepositoryList())), nil
}

func (r *RepositoryImpl) GetNamespaces(prefix string, closure bool) ([]string, error) {
	return cpi.FilterChildren(closure, prefix, r.getIndex().RepositoryList()), nil
}

////////////////////////////////////////////////////////////////////////////////
// forward

func (r *RepositoryImpl) IsReadOnly() bool {
	return r.base.IsReadOnly()
}

func (r *RepositoryImpl) Write(path string, mode vfs.FileMode, opts ...accessio.Option) error {
	return r.base.Write(path, mode, opts...)
}

func (r *RepositoryImpl) Update() error {
	return r.base.Update()
}

func (r *RepositoryImpl) Close() error {
	return r.base.Close()
}

func (a *RepositoryImpl) getIndex() *index.RepositoryIndex {
	if a.IsReadOnly() {
		return a.base.GetState().GetOriginalState().(*index.RepositoryIndex)
	}
	return a.base.GetState().GetState().(*index.RepositoryIndex)
}

////////////////////////////////////////////////////////////////////////////////
// cpi.Repository methods

func (r *RepositoryImpl) ExistsArtifact(name string, tag string) (bool, error) {
	return r.getIndex().HasArtifact(name, tag), nil
}

func (r *RepositoryImpl) LookupArtifact(name string, ref string) (acc cpi.ArtifactAccess, err error) {
	ns, err := NewNamespace(r, name)
	if err != nil {
		return nil, err
	}

	defer accessio.PropagateCloseTemporary(&err, ns) // temporary namespace object not exposed.

	a := r.getIndex().GetArtifactInfo(name, ref)
	if a == nil {
		return nil, cpi.ErrUnknownArtifact(name, ref)
	}
	return ns.GetArtifact(ref)
}

func (r *RepositoryImpl) LookupNamespace(name string) (cpi.NamespaceAccess, error) {
	return NewNamespace(r, name)
}
