// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package comparch

import (
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	ocicpi "github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localfsblob"
	ocmhdlr "github.com/open-component-model/ocm/pkg/contexts/ocm/blobhandler/handlers/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/repocpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/refmgmt"
)

////////////////////////////////////////////////////////////////////////////////

type _componentVersionAccess = cpi.ComponentVersionAccess

// ComponentArchive is the go representation for a component artifact.
type ComponentArchive struct {
	_componentVersionAccess
	spec      *RepositorySpec
	container *componentArchiveContainer
	main      cpi.Repository
	nonref    cpi.Repository
}

// New returns a new representation based element.
func New(ctx cpi.Context, acc accessobj.AccessMode, fs vfs.FileSystem, setup accessobj.Setup, closer accessobj.Closer, mode vfs.FileMode) (*ComponentArchive, error) {
	obj, err := accessobj.NewAccessObject(accessObjectInfo, acc, fs, setup, closer, mode)
	if err != nil {
		return nil, err
	}
	spec, err := NewRepositorySpec(acc, "")
	return _Wrap(ctx, obj, spec, err)
}

func _Wrap(ctx cpi.ContextProvider, obj *accessobj.AccessObject, spec *RepositorySpec, err error) (*ComponentArchive, error) {
	if err != nil {
		return nil, err
	}
	s := &componentArchiveContainer{
		ctx:   ctx.OCMContext(),
		fsacc: accessobj.NewFileSystemBlobAccess(obj),
		spec:  spec,
	}
	cv, err := repocpi.NewComponentVersionAccess(s.GetDescriptor().GetName(), s.GetDescriptor().GetVersion(), s, false, true, true)
	if err != nil {
		return nil, err
	}

	arch := &ComponentArchive{
		spec:      spec,
		container: s,
	}
	arch._componentVersionAccess = cv
	arch.main, arch.nonref = newRepository(arch)
	s.repo = arch.nonref
	return arch, nil
}

////////////////////////////////////////////////////////////////////////////////

var _ cpi.ComponentVersionAccess = &ComponentArchive{}

func (c *ComponentArchive) Close() error {
	return c.main.Close()
}

func (c *ComponentArchive) IsReadOnly() bool {
	return c.container.IsReadOnly()
}

// Repository returns a non referencing repository which does not
// close the archive.
func (c *ComponentArchive) Repository() cpi.Repository {
	return c.nonref
}

// AsRepository returns a repository view closing the archive.
func (c *ComponentArchive) AsRepository() cpi.Repository {
	return c.main
}

func (c *ComponentArchive) SetName(n string) {
	c.GetDescriptor().Name = n
}

func (c *ComponentArchive) SetVersion(v string) {
	c.GetDescriptor().Version = v
}

////////////////////////////////////////////////////////////////////////////////

type componentArchiveContainer struct {
	ctx   cpi.Context
	base  repocpi.ComponentVersionAccessBridge
	fsacc *accessobj.FileSystemBlobAccess
	spec  *RepositorySpec
	repo  cpi.Repository
}

var _ repocpi.ComponentVersionAccessImpl = (*componentArchiveContainer)(nil)

func (c *componentArchiveContainer) SetBridge(base repocpi.ComponentVersionAccessBridge) {
	c.base = base
}

func (c *componentArchiveContainer) GetParentBridge() repocpi.ComponentAccessBridge {
	return nil
}

func (c *componentArchiveContainer) Close() error {
	var list errors.ErrorList
	return list.Add(c.Update(), c.fsacc.Close()).Result()
}

func (c *componentArchiveContainer) GetContext() cpi.Context {
	return c.ctx
}

func (c *componentArchiveContainer) Repository() cpi.Repository {
	return c.repo
}

func (c *componentArchiveContainer) IsReadOnly() bool {
	return c.fsacc.IsReadOnly()
}

func (c *componentArchiveContainer) Update() error {
	return c.fsacc.Update()
}

func (c *componentArchiveContainer) SetDescriptor(cd *compdesc.ComponentDescriptor) error {
	if c.fsacc.IsReadOnly() {
		return accessobj.ErrReadOnly
	}
	cur := c.fsacc.GetState().GetState().(*compdesc.ComponentDescriptor)
	*cur = *cd.Copy()
	return c.fsacc.Update()
}

func (c *componentArchiveContainer) GetDescriptor() *compdesc.ComponentDescriptor {
	if c.fsacc.IsReadOnly() {
		return c.fsacc.GetState().GetOriginalState().(*compdesc.ComponentDescriptor)
	}
	return c.fsacc.GetState().GetState().(*compdesc.ComponentDescriptor)
}

func (c *componentArchiveContainer) GetBlob(name string) (cpi.DataAccess, error) {
	return c.fsacc.GetBlobDataByName(name)
}

func (c *componentArchiveContainer) GetStorageContext() cpi.StorageContext {
	return ocmhdlr.New(c.Repository(), c.base.GetName(), &BlobSink{c.fsacc}, Type)
}

type BlobSink struct {
	Sink ocicpi.BlobSink
}

func (s *BlobSink) AddBlob(blob blobaccess.BlobAccess) (string, error) {
	err := s.Sink.AddBlob(blob)
	if err != nil {
		return "", err
	}
	return blob.Digest().String(), nil
}

func (c *componentArchiveContainer) AddBlob(blob cpi.BlobAccess, refName string, global cpi.AccessSpec) (cpi.AccessSpec, error) {
	if blob == nil {
		return nil, errors.New("a resource has to be defined")
	}
	err := c.fsacc.AddBlob(blob)
	if err != nil {
		return nil, err
	}
	return localblob.New(common.DigestToFileName(blob.Digest()), refName, blob.MimeType(), global), nil
}

func (c *componentArchiveContainer) AccessMethod(a cpi.AccessSpec, cv refmgmt.ExtendedAllocatable) (cpi.AccessMethod, error) {
	if a.GetKind() == localblob.Type || a.GetKind() == localfsblob.Type {
		accessSpec, err := c.GetContext().AccessSpecForSpec(a)
		if err != nil {
			return nil, err
		}
		return newLocalFilesystemBlobAccessMethod(accessSpec.(*localblob.AccessSpec), c, cv)
	}
	return nil, errors.ErrNotSupported(errors.KIND_ACCESSMETHOD, a.GetType(), "component archive")
}

func (c *componentArchiveContainer) GetInexpensiveContentVersionIdentity(a cpi.AccessSpec, cv refmgmt.ExtendedAllocatable) string {
	if a.GetKind() == localblob.Type || a.GetKind() == localfsblob.Type {
		accessSpec, err := c.GetContext().AccessSpecForSpec(a)
		if err != nil {
			return ""
		}
		m, err := newLocalFilesystemBlobAccessMethod(accessSpec.(*localblob.AccessSpec), c, cv)
		if err != nil {
			return ""
		}
		defer m.Close()
		digest, _ := blobaccess.Digest(m)
		return digest.String()
	}
	return ""
}
