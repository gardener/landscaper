// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package support

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
)

type _ComponentVersionAccessImplBase = cpi.ComponentVersionAccessImplBase

type ComponentVersionAccessImpl interface {
	cpi.ComponentVersionAccessImpl
	EnablePersistence()
	Update(final bool) error
}

type componentVersionAccessImpl struct {
	*_ComponentVersionAccessImplBase
	lazy           bool
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
		discardChanges:                  !persistent,
		base:                            container,
	}
	container.SetImplementation(impl)
	return impl, nil
}

func (a *componentVersionAccessImpl) EnablePersistence() {
	a.discardChanges = false
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

func (a *componentVersionAccessImpl) SetResource(meta *cpi.ResourceMeta, acc compdesc.AccessSpec) error {
	res := &compdesc.Resource{
		ResourceMeta: *meta.Copy(),
		Access:       acc,
	}

	if res.Relation == metav1.LocalRelation {
		if res.Version == "" {
			res.Version = a.GetVersion()
		}
	}

	cd := a.GetDescriptor()
	if idx := cd.GetResourceIndex(meta); idx == -1 {
		cd.Resources = append(cd.Resources, *res)
		cd.Signatures = nil
	} else {
		if !cd.Resources[idx].ResourceMeta.HashEqual(&res.ResourceMeta) {
			cd.Signatures = nil
		}
		cd.Resources[idx] = *res
	}
	return a.Update(false)
}

func (a *componentVersionAccessImpl) SetSource(meta *cpi.SourceMeta, acc compdesc.AccessSpec) error {
	res := &compdesc.Source{
		SourceMeta: *meta.Copy(),
		Access:     acc,
	}

	if res.Version == "" {
		res.Version = a.GetVersion()
	}

	if idx := a.GetDescriptor().GetSourceIndex(meta); idx == -1 {
		a.GetDescriptor().Sources = append(a.GetDescriptor().Sources, *res)
	} else {
		a.GetDescriptor().Sources[idx] = *res
	}
	return a.Update(false)
}

func (a *componentVersionAccessImpl) SetReference(ref *cpi.ComponentReference) error {
	if idx := a.GetDescriptor().GetComponentReferenceIndex(*ref); idx == -1 {
		a.GetDescriptor().References = append(a.GetDescriptor().References, *ref)
	} else {
		a.GetDescriptor().References[idx] = *ref
	}
	return a.Update(false)
}

func (a *componentVersionAccessImpl) Update(final bool) error {
	if (final || !a.lazy) && !a.discardChanges {
		return a.base.Update()
	}
	return nil
}
