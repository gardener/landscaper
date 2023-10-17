// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

type DefaultStorageContext struct {
	ComponentRepository          Repository
	ComponentVersion             ComponentVersionAccess
	ImplementationRepositoryType ImplementationRepositoryType
}

var _ StorageContext = (*DefaultStorageContext)(nil)

func NewDefaultStorageContext(repo Repository, vers ComponentVersionAccess, reptype ImplementationRepositoryType) *DefaultStorageContext {
	return &DefaultStorageContext{
		ComponentRepository:          repo,
		ComponentVersion:             vers,
		ImplementationRepositoryType: reptype,
	}
}

func (c *DefaultStorageContext) GetContext() Context {
	return c.ComponentRepository.GetContext()
}

func (c *DefaultStorageContext) TargetComponentVersion() ComponentVersionAccess {
	return c.ComponentVersion
}

func (c *DefaultStorageContext) TargetComponentRepository() Repository {
	return c.ComponentRepository
}

func (c *DefaultStorageContext) GetImplementationRepositoryType() ImplementationRepositoryType {
	return c.ImplementationRepositoryType
}
