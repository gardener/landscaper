// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package comparch

import (
	"fmt"
	"strings"
	"sync"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/attrs/vfsattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localfsblob"
	ocmhdlr "github.com/open-component-model/ocm/pkg/contexts/ocm/blobhandler/handlers/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/repocpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/refmgmt"
	"github.com/open-component-model/ocm/pkg/utils"
)

type RepositoryImpl struct {
	lock   sync.RWMutex
	bridge repocpi.RepositoryBridge
	arch   *ComponentArchive
	nonref cpi.Repository
}

var _ repocpi.RepositoryImpl = (*RepositoryImpl)(nil)

func NewRepository(ctx cpi.Context, s *RepositorySpec) (cpi.Repository, error) {
	if s.GetPathFileSystem() == nil {
		s.SetPathFileSystem(vfsattr.Get(ctx))
	}
	a, err := Open(ctx, s.AccessMode, s.FilePath, 0o700, s)
	if err != nil {
		return nil, err
	}
	return a.AsRepository(), nil
}

func newRepository(a *ComponentArchive) (main, nonref cpi.Repository) {
	// close main cv abstraction on repository close -------v
	impl := &RepositoryImpl{
		arch: a,
	}
	r := repocpi.NewRepository(impl, "comparch")
	return r, impl.nonref
}

func (r *RepositoryImpl) Close() error {
	return r.arch.container.Close()
}

func (r *RepositoryImpl) SetBridge(base repocpi.RepositoryBridge) {
	r.bridge = base
	r.nonref = repocpi.NewNoneRefRepositoryView(base)
}

func (r *RepositoryImpl) GetContext() cpi.Context {
	return r.arch.GetContext()
}

func (r *RepositoryImpl) ComponentLister() cpi.ComponentLister {
	return r
}

func (r *RepositoryImpl) matchPrefix(prefix string, closure bool) bool {
	if r.arch.GetName() != prefix {
		if prefix != "" && !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		if !closure || !strings.HasPrefix(r.arch.GetName(), prefix) {
			return false
		}
	}
	return true
}

func (r *RepositoryImpl) NumComponents(prefix string) (int, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if r.arch == nil {
		return -1, accessio.ErrClosed
	}
	if !r.matchPrefix(prefix, true) {
		return 0, nil
	}
	return 1, nil
}

func (r *RepositoryImpl) GetComponents(prefix string, closure bool) ([]string, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if r.arch == nil {
		return nil, accessio.ErrClosed
	}
	if !r.matchPrefix(prefix, closure) {
		return []string{}, nil
	}
	return []string{r.arch.GetName()}, nil
}

func (r *RepositoryImpl) Get() *ComponentArchive {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if r.arch != nil {
		return r.arch
	}
	return nil
}

func (r *RepositoryImpl) GetSpecification() cpi.RepositorySpec {
	return r.arch.spec
}

func (r *RepositoryImpl) ExistsComponentVersion(name string, ref string) (bool, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if r.arch == nil {
		return false, accessio.ErrClosed
	}
	return r.arch.GetName() == name && r.arch.GetVersion() == ref, nil
}

func (r *RepositoryImpl) LookupComponent(name string) (*repocpi.ComponentAccessInfo, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if r.arch == nil {
		return nil, accessio.ErrClosed
	}
	if r.arch.GetName() != name {
		return nil, errors.ErrNotFound(errors.KIND_COMPONENT, name, Type)
	}
	return newComponentAccess(r)
}

////////////////////////////////////////////////////////////////////////////////

type ComponentAccessImpl struct {
	base repocpi.ComponentAccessBridge
	repo *RepositoryImpl
}

var _ repocpi.ComponentAccessImpl = (*ComponentAccessImpl)(nil)

func newComponentAccess(r *RepositoryImpl) (*repocpi.ComponentAccessInfo, error) {
	impl := &ComponentAccessImpl{
		repo: r,
	}
	return &repocpi.ComponentAccessInfo{impl, "component archive", true}, nil
}

func (c *ComponentAccessImpl) Close() error {
	return nil
}

func (c *ComponentAccessImpl) SetBridge(base repocpi.ComponentAccessBridge) {
	c.base = base
}

func (c *ComponentAccessImpl) GetParentBridge() repocpi.RepositoryViewManager {
	return c.repo.bridge
}

func (c *ComponentAccessImpl) GetContext() cpi.Context {
	return c.repo.GetContext()
}

func (c *ComponentAccessImpl) GetName() string {
	return c.repo.arch.GetName()
}

func (c *ComponentAccessImpl) IsReadOnly() bool {
	return c.repo.arch.IsReadOnly()
}

func (c *ComponentAccessImpl) ListVersions() ([]string, error) {
	return []string{c.repo.arch.GetVersion()}, nil
}

func (c *ComponentAccessImpl) HasVersion(vers string) (bool, error) {
	return vers == c.repo.arch.GetVersion(), nil
}

func (c *ComponentAccessImpl) LookupVersion(version string) (*repocpi.ComponentVersionAccessInfo, error) {
	if version != c.repo.arch.GetVersion() {
		return nil, errors.ErrNotFound(cpi.KIND_COMPONENTVERSION, fmt.Sprintf("%s:%s", c.GetName(), c.repo.arch.GetVersion()))
	}
	return newComponentVersionAccess(c, version, false)
}

func (c *ComponentAccessImpl) NewVersion(version string, overrides ...bool) (*repocpi.ComponentVersionAccessInfo, error) {
	if version != c.repo.arch.GetVersion() {
		return nil, errors.ErrNotSupported(cpi.KIND_COMPONENTVERSION, version, fmt.Sprintf("component archive %s:%s", c.GetName(), c.repo.arch.GetVersion()))
	}
	if !utils.Optional(overrides...) {
		return nil, errors.ErrAlreadyExists(cpi.KIND_COMPONENTVERSION, fmt.Sprintf("%s:%s", c.GetName(), c.repo.arch.GetVersion()))
	}
	return newComponentVersionAccess(c, version, false)
}

////////////////////////////////////////////////////////////////////////////////

type ComponentVersionContainer struct {
	impl repocpi.ComponentVersionAccessBridge

	comp *ComponentAccessImpl

	descriptor *compdesc.ComponentDescriptor
}

var _ repocpi.ComponentVersionAccessImpl = (*ComponentVersionContainer)(nil)

func newComponentVersionAccess(comp *ComponentAccessImpl, version string, persistent bool) (*repocpi.ComponentVersionAccessInfo, error) {
	c, err := newComponentVersionContainer(comp)
	if err != nil {
		return nil, err
	}
	return &repocpi.ComponentVersionAccessInfo{c, true, persistent}, nil
}

func newComponentVersionContainer(comp *ComponentAccessImpl) (*ComponentVersionContainer, error) {
	return &ComponentVersionContainer{
		comp:       comp,
		descriptor: comp.repo.arch.GetDescriptor(),
	}, nil
}

func (c *ComponentVersionContainer) SetBridge(impl repocpi.ComponentVersionAccessBridge) {
	c.impl = impl
}

func (c *ComponentVersionContainer) GetParentBridge() repocpi.ComponentAccessBridge {
	return c.comp.base
}

func (c *ComponentVersionContainer) Close() error {
	return nil
}

func (c *ComponentVersionContainer) GetContext() cpi.Context {
	return c.comp.GetContext()
}

func (c *ComponentVersionContainer) Repository() cpi.Repository {
	return c.comp.repo.arch.nonref
}

func (c *ComponentVersionContainer) IsReadOnly() bool {
	return c.comp.repo.arch.IsReadOnly()
}

func (c *ComponentVersionContainer) Update() error {
	desc := c.comp.repo.arch.GetDescriptor()
	*desc = *c.descriptor.Copy()
	return c.comp.repo.arch.container.Update()
}

func (c *ComponentVersionContainer) SetDescriptor(cd *compdesc.ComponentDescriptor) error {
	*c.descriptor = *cd
	return c.Update()
}

func (c *ComponentVersionContainer) GetDescriptor() *compdesc.ComponentDescriptor {
	return c.descriptor
}

func (c *ComponentVersionContainer) GetBlob(name string) (cpi.DataAccess, error) {
	return c.comp.repo.arch.container.GetBlob(name)
}

func (c *ComponentVersionContainer) GetStorageContext() cpi.StorageContext {
	return ocmhdlr.New(c.Repository(), c.comp.GetName(), &BlobSink{c.comp.repo.arch.container.fsacc}, Type)
}

func (c *ComponentVersionContainer) AddBlob(blob cpi.BlobAccess, refName string, global cpi.AccessSpec) (cpi.AccessSpec, error) {
	if blob == nil {
		return nil, errors.New("a resource has to be defined")
	}
	err := c.comp.repo.arch.container.fsacc.AddBlob(blob)
	if err != nil {
		return nil, err
	}
	return localblob.New(common.DigestToFileName(blob.Digest()), refName, blob.MimeType(), global), nil
}

func (c *ComponentVersionContainer) AccessMethod(a cpi.AccessSpec, cv refmgmt.ExtendedAllocatable) (cpi.AccessMethod, error) {
	if a.GetKind() == localblob.Type || a.GetKind() == localfsblob.Type {
		accessSpec, err := c.GetContext().AccessSpecForSpec(a)
		if err != nil {
			return nil, err
		}
		return newLocalFilesystemBlobAccessMethod(accessSpec.(*localblob.AccessSpec), c, cv)
	}
	return nil, errors.ErrNotSupported(errors.KIND_ACCESSMETHOD, a.GetType(), "component archive")
}

func (c *ComponentVersionContainer) GetInexpensiveContentVersionIdentity(a cpi.AccessSpec, cv refmgmt.ExtendedAllocatable) string {
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
