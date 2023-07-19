// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package comparch

import (
	"io"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	ocicpi "github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localfsblob"
	ocmhdlr "github.com/open-component-model/ocm/pkg/contexts/ocm/blobhandler/handlers/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/support"
	"github.com/open-component-model/ocm/pkg/errors"
)

////////////////////////////////////////////////////////////////////////////////

// ComponentArchive is the go representation for a component artifact.
type ComponentArchive struct {
	spec      *RepositorySpec
	container *componentArchiveContainer
	main      io.Closer
	nonref    cpi.Repository
	cpi.ComponentVersionAccess
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
		ctx:  ctx.OCMContext(),
		base: accessobj.NewFileSystemBlobAccess(obj),
	}
	impl, err := support.NewComponentVersionAccessImpl(s.GetDescriptor().GetName(), s.GetDescriptor().GetVersion(), s, false, true)
	if err != nil {
		return nil, err
	}
	s.spec = spec
	arch := &ComponentArchive{
		spec:      spec,
		container: s,
	}
	arch.ComponentVersionAccess = cpi.NewComponentVersionAccess(impl)
	arch.main, arch.nonref = newRepository(arch)
	s.repo = arch.nonref
	return arch, nil
}

////////////////////////////////////////////////////////////////////////////////

var _ cpi.ComponentVersionAccess = &ComponentArchive{}

func (c *ComponentArchive) Close() error {
	return c.main.Close()
}

func (c *ComponentArchive) Repository() cpi.Repository {
	return c.nonref
}

func (c *ComponentArchive) SetName(n string) {
	c.GetDescriptor().Name = n
}

func (c *ComponentArchive) SetVersion(v string) {
	c.GetDescriptor().Version = v
}

////////////////////////////////////////////////////////////////////////////////

type componentArchiveContainer struct {
	ctx  cpi.Context
	impl support.ComponentVersionAccessImpl
	base *accessobj.FileSystemBlobAccess
	spec *RepositorySpec
	repo cpi.Repository
}

var _ support.ComponentVersionContainer = (*componentArchiveContainer)(nil)

func (c *componentArchiveContainer) SetImplementation(impl support.ComponentVersionAccessImpl) {
	c.impl = impl
}

func (c *componentArchiveContainer) GetParentViewManager() cpi.ComponentAccessViewManager {
	return nil
}

func (c *componentArchiveContainer) Close() error {
	c.Update()
	return c.base.Close()
}

func (c *componentArchiveContainer) GetContext() cpi.Context {
	return c.ctx
}

func (c *componentArchiveContainer) Repository() cpi.Repository {
	return c.repo
}

func (c *componentArchiveContainer) IsReadOnly() bool {
	return c.base.IsReadOnly()
}

func (c *componentArchiveContainer) Update() error {
	return c.base.Update()
}

func (c *componentArchiveContainer) GetDescriptor() *compdesc.ComponentDescriptor {
	if c.base.IsReadOnly() {
		return c.base.GetState().GetOriginalState().(*compdesc.ComponentDescriptor)
	}
	return c.base.GetState().GetState().(*compdesc.ComponentDescriptor)
}

func (c *componentArchiveContainer) GetBlobData(name string) (cpi.DataAccess, error) {
	return c.base.GetBlobDataByName(name)
}

func (c *componentArchiveContainer) GetStorageContext(cv cpi.ComponentVersionAccess) cpi.StorageContext {
	return ocmhdlr.New(c.Repository(), cv, &BlobSink{c.base}, Type)
}

type BlobSink struct {
	Sink ocicpi.BlobSink
}

func (s *BlobSink) AddBlob(blob accessio.BlobAccess) (string, error) {
	err := s.Sink.AddBlob(blob)
	if err != nil {
		return "", err
	}
	return blob.Digest().String(), nil
}

func (c *componentArchiveContainer) AddBlobFor(storagectx cpi.StorageContext, blob cpi.BlobAccess, refName string, global cpi.AccessSpec) (cpi.AccessSpec, error) {
	if blob == nil {
		return nil, errors.New("a resource has to be defined")
	}
	err := c.base.AddBlob(blob)
	if err != nil {
		return nil, err
	}
	return localblob.New(common.DigestToFileName(blob.Digest()), refName, blob.MimeType(), global), nil
}

func (c *componentArchiveContainer) AccessMethod(a cpi.AccessSpec) (cpi.AccessMethod, error) {
	if a.GetKind() == localblob.Type || a.GetKind() == localfsblob.Type {
		accessSpec, err := c.GetContext().AccessSpecForSpec(a)
		if err != nil {
			return nil, err
		}
		return newLocalFilesystemBlobAccessMethod(accessSpec.(*localblob.AccessSpec), c), nil
	}
	return nil, errors.ErrNotSupported(errors.KIND_ACCESSMETHOD, a.GetType(), "component archive")
}

func (c *componentArchiveContainer) GetInexpensiveContentVersionIdentity(a cpi.AccessSpec) string {
	if a.GetKind() == localblob.Type || a.GetKind() == localfsblob.Type {
		accessSpec, err := c.GetContext().AccessSpecForSpec(a)
		if err != nil {
			return ""
		}
		digest, _ := accessio.Digest(newLocalFilesystemBlobAccessMethod(accessSpec.(*localblob.AccessSpec), c))
		return digest.String()
	}
	return ""
}
