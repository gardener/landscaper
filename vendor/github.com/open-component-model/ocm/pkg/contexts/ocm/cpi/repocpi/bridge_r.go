// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package repocpi

import (
	"io"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/refmgmt"
	"github.com/open-component-model/ocm/pkg/refmgmt/resource"
)

type ComponentAccessInfo struct {
	Impl ComponentAccessImpl
	Kind string
	Main bool
}

type RepositoryImpl interface {
	SetBridge(bridge RepositoryBridge)

	GetContext() cpi.Context

	GetSpecification() cpi.RepositorySpec
	ComponentLister() cpi.ComponentLister

	ExistsComponentVersion(name string, version string) (bool, error)
	LookupComponent(name string) (*ComponentAccessInfo, error)

	io.Closer
}

type _repositoryBridgeBase = resource.ResourceImplBase[cpi.Repository]

type repositoryBridge struct {
	*_repositoryBridgeBase
	ctx  cpi.Context
	kind string
	impl RepositoryImpl
}

func newRepositoryBridge(impl RepositoryImpl, kind string, closer ...io.Closer) RepositoryBridge {
	base := resource.NewSimpleResourceImplBase[cpi.Repository](closer...)
	b := &repositoryBridge{
		_repositoryBridgeBase: base,
		ctx:                   impl.GetContext(),
		impl:                  impl,
	}
	impl.SetBridge(b)
	return b
}

func (b *repositoryBridge) Close() error {
	list := errors.ErrListf("closing %s", b.kind)
	refmgmt.AllocLog.Trace("closing repository bridge", "kind", b.kind)
	list.Add(b.impl.Close())
	list.Add(b._repositoryBridgeBase.Close())
	refmgmt.AllocLog.Trace("closed repository bridge", "kind", b.kind)
	return list.Result()
}

func (b *repositoryBridge) GetContext() cpi.Context {
	return b.ctx
}

func (b *repositoryBridge) GetSpecification() cpi.RepositorySpec {
	return b.impl.GetSpecification()
}

func (b *repositoryBridge) ComponentLister() cpi.ComponentLister {
	return b.impl.ComponentLister()
}

func (b *repositoryBridge) ExistsComponentVersion(name string, version string) (bool, error) {
	return b.impl.ExistsComponentVersion(name, version)
}

func (b *repositoryBridge) LookupComponentVersion(name string, version string) (cv cpi.ComponentVersionAccess, rerr error) {
	c, err := b.LookupComponent(name)
	if err != nil {
		return nil, err
	}
	defer refmgmt.PropagateCloseTemporary(&rerr, c) // temporary component object not exposed.
	refmgmt.AllocLog.Trace("lookup version for temporary component ref", "component", name, "version", version)
	return c.LookupVersion(version)
}

func (b *repositoryBridge) LookupComponent(name string) (cpi.ComponentAccess, error) {
	i, err := b.impl.LookupComponent(name)
	if err != nil {
		return nil, err
	}
	return NewComponentAccess(i.Impl, i.Kind, i.Main)
}
