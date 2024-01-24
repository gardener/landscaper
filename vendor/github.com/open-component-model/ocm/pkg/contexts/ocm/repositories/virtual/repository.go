// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package virtual

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/repocpi"
)

type RepositoryImpl struct {
	bridge repocpi.RepositoryBridge
	ctx    cpi.Context
	access Access
	nonref cpi.Repository
}

var _ repocpi.RepositoryImpl = (*RepositoryImpl)(nil)

func NewRepository(ctx cpi.Context, acc Access) cpi.Repository {
	impl := &RepositoryImpl{
		ctx:    ctx,
		access: acc,
	}
	return repocpi.NewRepository(impl, "OCM repo[Simple]")
}

func (r *RepositoryImpl) Close() error {
	return r.access.Close()
}

func (r *RepositoryImpl) SetBridge(base repocpi.RepositoryBridge) {
	r.bridge = base
	r.nonref = repocpi.NewNoneRefRepositoryView(base)
}

func (r *RepositoryImpl) GetContext() cpi.Context {
	return r.ctx
}

func (r *RepositoryImpl) GetSpecification() cpi.RepositorySpec {
	if p, ok := r.access.(RepositorySpecProvider); ok {
		return p.GetSpecification()
	}
	return NewRepositorySpec(r.access)
}

func (r *RepositoryImpl) ComponentLister() cpi.ComponentLister {
	return r.access.ComponentLister()
}

func (r *RepositoryImpl) ExistsComponentVersion(name string, version string) (bool, error) {
	return r.access.ExistsComponentVersion(name, version)
}

func (r *RepositoryImpl) LookupComponent(name string) (*repocpi.ComponentAccessInfo, error) {
	return newComponentAccess(r, name, true)
}
