// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
)

////////////////////////////////////////////////////////////////////////////////

type BaseAccess struct {
	vers   ComponentVersionAccess
	access compdesc.AccessSpec
}

type baseAccess = BaseAccess

func NewBaseAccess(cv ComponentVersionAccess, acc compdesc.AccessSpec) *BaseAccess {
	return &BaseAccess{vers: cv, access: acc}
}

func (r *BaseAccess) ComponentVersion() ComponentVersionAccess {
	return r.vers
}

func (r *BaseAccess) Access() (AccessSpec, error) {
	return r.vers.GetContext().AccessSpecForSpec(r.access)
}

func (r *BaseAccess) AccessMethod() (AccessMethod, error) {
	acc, err := r.vers.GetContext().AccessSpecForSpec(r.access)
	if err != nil {
		return nil, err
	}
	return r.vers.AccessMethod(acc)
}

////////////////////////////////////////////////////////////////////////////////

type ResourceAccessImpl struct {
	*baseAccess
	meta ResourceMeta
}

var _ ResourceAccess = (*ResourceAccessImpl)(nil)

func newResourceAccess(componentVersion ComponentVersionAccess, accessSpec compdesc.AccessSpec, meta ResourceMeta) *ResourceAccessImpl {
	return &ResourceAccessImpl{
		baseAccess: NewBaseAccess(componentVersion, accessSpec),
		meta:       meta,
	}
}

func (r *ResourceAccessImpl) Meta() *ResourceMeta {
	return &r.meta
}

////////////////////////////////////////////////////////////////////////////////

type SourceAccessImpl struct {
	*baseAccess
	meta SourceMeta
}

var _ SourceAccess = (*SourceAccessImpl)(nil)

func newSourceAccess(componentVersion ComponentVersionAccess, accessSpec compdesc.AccessSpec, meta SourceMeta) *SourceAccessImpl {
	return &SourceAccessImpl{
		baseAccess: NewBaseAccess(componentVersion, accessSpec),
		meta:       meta,
	}
}

func (r SourceAccessImpl) Meta() *SourceMeta {
	return &r.meta
}
