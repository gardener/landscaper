// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

type DefaultStorageContext struct {
	ComponentRepository          Repository
	ComponentName                string
	ImplementationRepositoryType ImplementationRepositoryType
}

var _ StorageContext = (*DefaultStorageContext)(nil)

func NewDefaultStorageContext(repo Repository, compname string, reptype ImplementationRepositoryType) *DefaultStorageContext {
	return &DefaultStorageContext{
		ComponentRepository:          repo,
		ComponentName:                compname,
		ImplementationRepositoryType: reptype,
	}
}

func (c *DefaultStorageContext) GetContext() Context {
	return c.ComponentRepository.GetContext()
}

func (c *DefaultStorageContext) TargetComponentName() string {
	return c.ComponentName
}

func (c *DefaultStorageContext) TargetComponentRepository() Repository {
	return c.ComponentRepository
}

func (c *DefaultStorageContext) GetImplementationRepositoryType() ImplementationRepositoryType {
	return c.ImplementationRepositoryType
}
