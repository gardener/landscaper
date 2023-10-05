// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package support

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
)

type _ComponentVersionAccessImplBase = cpi.ComponentVersionAccessImplBase

type ComponentVersionAccessImpl interface {
	cpi.ComponentVersionAccessImpl
	EnablePersistence() bool
	Update(final bool) error
}

type componentVersionAccessImpl struct {
	*_ComponentVersionAccessImplBase
	lazy           bool
	persistent     bool
	discardChanges bool
	base           ComponentVersionContainer
}

var _ ComponentVersionAccessImpl = (*componentVersionAccessImpl)(nil)

func GetComponentVersionContainer(cv cpi.ComponentVersionAccess) (ComponentVersionContainer, error) {
	impl, err := cpi.GetComponentVersionAccessImplementation(cv)
	if err != nil {
		return nil, err
	}
	if mine, ok := impl.(*componentVersionAccessImpl); ok {
		return mine.base, nil
	}
	return nil, errors.Newf("non-matching component version implementation %T", impl)
}

func NewComponentVersionAccessImpl(name, version string, container ComponentVersionContainer, lazy bool, persistent bool) (cpi.ComponentVersionAccessImpl, error) {
	base, err := cpi.NewComponentVersionAccessImplBase(container.GetContext(), name, version, container.GetParentViewManager())
	if err != nil {
		return nil, err
	}
	impl := &componentVersionAccessImpl{
		_ComponentVersionAccessImplBase: base,
		lazy:                            lazy,
		persistent:                      persistent,
		base:                            container,
	}
	container.SetImplementation(impl)
	return impl, nil
}

func (a *componentVersionAccessImpl) EnablePersistence() bool {
	if a.discardChanges {
		return false
	}
	a.persistent = true
	return true
}

func (a *componentVersionAccessImpl) IsPersistent() bool {
	return a.persistent
}

func (a *componentVersionAccessImpl) DiscardChanges() {
	a.discardChanges = true
}

func (a *componentVersionAccessImpl) Close() error {
	return errors.ErrListf("closing component version access %s/%s", a.GetName(), a.GetVersion()).Add(a.Update(true), a.base.Close(), a._ComponentVersionAccessImplBase.Close()).Result()
}

func (a *componentVersionAccessImpl) Repository() cpi.Repository {
	return a.base.Repository()
}

func (a *componentVersionAccessImpl) IsReadOnly() bool {
	return a.base.IsReadOnly()
}

////////////////////////////////////////////////////////////////////////////////
// with access to actual view

func (a *componentVersionAccessImpl) AccessMethod(cv cpi.ComponentVersionAccess, acc cpi.AccessSpec) (cpi.AccessMethod, error) {
	return a.base.AccessMethod(acc)
}

func (a *componentVersionAccessImpl) GetInexpensiveContentVersionIdentity(cv cpi.ComponentVersionAccess, acc cpi.AccessSpec) string {
	return a.base.GetInexpensiveContentVersionIdentity(acc)
}

func (a *componentVersionAccessImpl) GetDescriptor() *compdesc.ComponentDescriptor {
	return a.base.GetDescriptor()
}

func (a *componentVersionAccessImpl) GetStorageContext(cv cpi.ComponentVersionAccess) cpi.StorageContext {
	return a.base.GetStorageContext(cv)
}

func (a *componentVersionAccessImpl) AddBlobFor(storagectx cpi.StorageContext, blob cpi.BlobAccess, refName string, global cpi.AccessSpec) (cpi.AccessSpec, error) {
	return a.base.AddBlobFor(storagectx, blob, refName, global)
}

func (a *componentVersionAccessImpl) Update(final bool) error {
	if (final || !a.lazy) && !a.discardChanges && a.persistent {
		return a.base.Update()
	}
	return nil
}
