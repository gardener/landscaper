// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package virtual

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/repocpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
)

type componentAccessImpl struct {
	bridge repocpi.ComponentAccessBridge

	repo *RepositoryImpl
	name string
}

var _ repocpi.ComponentAccessImpl = (*componentAccessImpl)(nil)

func newComponentAccess(repo *RepositoryImpl, name string, main bool) (*repocpi.ComponentAccessInfo, error) {
	impl := &componentAccessImpl{
		repo: repo,
		name: name,
	}
	return &repocpi.ComponentAccessInfo{impl, "OCM component[Simple]", main}, nil
}

func (c *componentAccessImpl) Close() error {
	return nil
}

func (c *componentAccessImpl) SetBridge(base repocpi.ComponentAccessBridge) {
	c.bridge = base
}

func (c *componentAccessImpl) GetParentBridge() repocpi.RepositoryViewManager {
	return c.repo.bridge
}

func (c *componentAccessImpl) GetContext() cpi.Context {
	return c.repo.GetContext()
}

func (c *componentAccessImpl) GetName() string {
	return c.name
}

func (c *componentAccessImpl) ListVersions() ([]string, error) {
	return c.repo.access.ListVersions(c.name)
}

func (c *componentAccessImpl) HasVersion(vers string) (bool, error) {
	return c.repo.ExistsComponentVersion(c.name, vers)
}

func (c *componentAccessImpl) IsReadOnly() bool {
	return c.repo.access.IsReadOnly()
}

func (c *componentAccessImpl) LookupVersion(version string) (*repocpi.ComponentVersionAccessInfo, error) {
	ok, err := c.HasVersion(version)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, cpi.ErrComponentVersionNotFoundWrap(err, c.name, version)
	}
	return newComponentVersionAccess(c, version, true)
}

func (c *componentAccessImpl) NewVersion(version string, overrides ...bool) (*repocpi.ComponentVersionAccessInfo, error) {
	override := utils.Optional(overrides...)
	ok, err := c.HasVersion(version)
	if err == nil && ok {
		if override {
			return newComponentVersionAccess(c, version, false)
		}
		return nil, errors.ErrAlreadyExists(cpi.KIND_COMPONENTVERSION, c.name+"/"+version)
	}
	if err != nil && !errors.IsErrNotFoundKind(err, cpi.KIND_COMPONENTVERSION) {
		return nil, err
	}
	return newComponentVersionAccess(c, version, false)
}
