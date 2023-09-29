// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package comparch

import (
	"strings"
	"sync"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/attrs/vfsattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
)

type _RepositoryImplBase = cpi.RepositoryImplBase

type RepositoryImpl struct {
	_RepositoryImplBase
	lock sync.RWMutex
	arch *ComponentArchive
}

var _ cpi.RepositoryImpl = (*RepositoryImpl)(nil)

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

func newRepository(a *ComponentArchive) (main cpi.Repository, nonref cpi.Repository) {
	base := cpi.NewRepositoryImplBase(a.GetContext(), a.ComponentVersionAccess)
	impl := &RepositoryImpl{
		_RepositoryImplBase: *base,
		arch:                a,
	}
	return cpi.NewRepository(impl), cpi.NewNoneRefRepositoryView(impl)
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

func (r *RepositoryImpl) LookupComponentVersion(name string, version string) (cpi.ComponentVersionAccess, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	ok, err := r.ExistsComponentVersion(name, version)
	if ok {
		return r.arch.Dup()
	}
	if err == nil {
		err = errors.ErrNotFound(cpi.KIND_COMPONENTVERSION, common.NewNameVersion(name, version).String(), Type)
	}
	return nil, err
}

func (r *RepositoryImpl) LookupComponent(name string) (cpi.ComponentAccess, error) {
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

type _ComponentAccessImplBase = cpi.ComponentAccessImplBase

type ComponentAccessImpl struct {
	_ComponentAccessImplBase
	repo *RepositoryImpl
}

var _ cpi.ComponentAccessImpl = (*ComponentAccessImpl)(nil)

func newComponentAccess(r *RepositoryImpl) (cpi.ComponentAccess, error) {
	base, err := cpi.NewComponentAccessImplBase(r.GetContext(), r.arch.GetName(), r)
	if err != nil {
		return nil, err
	}
	impl := &ComponentAccessImpl{
		_ComponentAccessImplBase: *base,
		repo:                     r,
	}
	return cpi.NewComponentAccess(impl, "component archive"), nil
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

func (c *ComponentAccessImpl) LookupVersion(ref string) (cpi.ComponentVersionAccess, error) {
	return c.repo.LookupComponentVersion(c.repo.arch.GetName(), ref)
}

func (c *ComponentAccessImpl) AddVersion(access cpi.ComponentVersionAccess) error {
	return errors.ErrNotSupported(errors.KIND_FUNCTION, "add version", Type)
}

func (c *ComponentAccessImpl) NewVersion(version string, overrides ...bool) (cpi.ComponentVersionAccess, error) {
	return nil, errors.ErrNotSupported(errors.KIND_FUNCTION, "new version", Type)
}
