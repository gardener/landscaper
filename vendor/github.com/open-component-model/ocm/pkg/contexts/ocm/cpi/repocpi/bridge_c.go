// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package repocpi

import (
	"io"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/compositionmodeattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/finalizer"
	"github.com/open-component-model/ocm/pkg/optionutils"
	"github.com/open-component-model/ocm/pkg/refmgmt"
	"github.com/open-component-model/ocm/pkg/refmgmt/resource"
)

type ComponentVersionAccessInfo struct {
	Impl       ComponentVersionAccessImpl
	Lazy       bool
	Persistent bool
}

// ComponentAccessImpl is the provider implementation
// interface for component versions.
type ComponentAccessImpl interface {
	SetBridge(bridge ComponentAccessBridge)
	GetParentBridge() RepositoryViewManager

	GetContext() cpi.Context
	GetName() string
	IsReadOnly() bool

	ListVersions() ([]string, error)
	HasVersion(vers string) (bool, error)
	LookupVersion(version string) (*ComponentVersionAccessInfo, error)
	NewVersion(version string, overrides ...bool) (*ComponentVersionAccessInfo, error)

	io.Closer
}

type _componentAccessBridgeBase = resource.ResourceImplBase[cpi.ComponentAccess]

type componentAccessBridge struct {
	*_componentAccessBridgeBase
	ctx  cpi.Context
	name string
	impl ComponentAccessImpl
}

func newComponentAccessBridge(impl ComponentAccessImpl, closer ...io.Closer) (ComponentAccessBridge, error) {
	base, err := resource.NewResourceImplBase[cpi.ComponentAccess, cpi.Repository](impl.GetParentBridge(), closer...)
	if err != nil {
		return nil, err
	}
	b := &componentAccessBridge{
		_componentAccessBridgeBase: base,
		ctx:                        impl.GetContext(),
		name:                       impl.GetName(),
		impl:                       impl,
	}
	impl.SetBridge(b)
	return b, nil
}

func (b *componentAccessBridge) Close() error {
	list := errors.ErrListf("closing component %s", b.name)
	refmgmt.AllocLog.Trace("closing component bridge", "name", b.name)
	list.Add(b.impl.Close())
	list.Add(b._componentAccessBridgeBase.Close())
	refmgmt.AllocLog.Trace("closed component bridge", "name", b.name)
	return list.Result()
}

func (b *componentAccessBridge) GetContext() cpi.Context {
	return b.ctx
}

func (b *componentAccessBridge) GetName() string {
	return b.name
}

func (b *componentAccessBridge) IsReadOnly() bool {
	return b.impl.IsReadOnly()
}

func (c *componentAccessBridge) IsOwned(cv cpi.ComponentVersionAccess) bool {
	bridge, err := GetComponentVersionAccessBridge(cv)
	if err != nil {
		return false
	}

	impl := bridge.(*componentVersionAccessBridge).impl
	cvcompmgr := impl.GetParentBridge()
	return c == cvcompmgr
}

func (b *componentAccessBridge) ListVersions() ([]string, error) {
	return b.impl.ListVersions()
}

func (b *componentAccessBridge) LookupVersion(version string) (cpi.ComponentVersionAccess, error) {
	i, err := b.impl.LookupVersion(version)
	if err != nil {
		return nil, err
	}
	if i == nil || i.Impl == nil {
		return nil, errors.ErrInvalid("component implementation behaviour", "LookupVersion")
	}
	return NewComponentVersionAccess(b.GetName(), version, i.Impl, i.Lazy, i.Persistent, !compositionmodeattr.Get(b.GetContext()))
}

func (b *componentAccessBridge) HasVersion(vers string) (bool, error) {
	return b.impl.HasVersion(vers)
}

func (b *componentAccessBridge) NewVersion(version string, overrides ...bool) (cpi.ComponentVersionAccess, error) {
	i, err := b.impl.NewVersion(version, overrides...)
	if err != nil {
		return nil, err
	}
	if i == nil || i.Impl == nil {
		return nil, errors.ErrInvalid("component implementation behaviour", "NewVersion")
	}
	return NewComponentVersionAccess(b.GetName(), version, i.Impl, i.Lazy, false, !compositionmodeattr.Get(b.GetContext()))
}

func (c *componentAccessBridge) AddVersion(cv cpi.ComponentVersionAccess, opts *cpi.AddVersionOptions) (ferr error) {
	var finalize finalizer.Finalizer
	defer finalize.FinalizeWithErrorPropagation(&ferr)

	cvbridge, err := GetComponentVersionAccessBridge(cv)
	if err != nil {
		return err
	}

	forcestore := c.IsOwned(cv)
	if !forcestore {
		eff, err := c.NewVersion(cv.GetVersion(), optionutils.AsValue(opts.Overwrite))
		if err != nil {
			return err
		}
		finalize.With(func() error {
			return eff.Close()
		})
		cvbridge, err = GetComponentVersionAccessBridge(eff)
		if err != nil {
			return err
		}

		d := eff.GetDescriptor()
		*d = *cv.GetDescriptor().Copy()

		err = c.setupLocalBlobs("resource", cv, cvbridge, d.Resources, &opts.BlobUploadOptions)
		if err == nil {
			err = c.setupLocalBlobs("source", cv, cvbridge, d.Sources, &opts.BlobUploadOptions)
		}
		if err != nil {
			return err
		}
	}
	cvbridge.EnablePersistence()
	err = cvbridge.Update(!cvbridge.UseDirectAccess())
	return err
}

func (c *componentAccessBridge) setupLocalBlobs(kind string, src cpi.ComponentVersionAccess, tgtbridge ComponentVersionAccessBridge, it compdesc.ArtifactAccessor, opts *cpi.BlobUploadOptions) (ferr error) {
	ctx := src.GetContext()
	// transfer all local blobs
	prov := func(spec cpi.AccessSpec) (blob blobaccess.BlobAccess, ref string, global cpi.AccessSpec, err error) {
		if spec.IsLocal(ctx) {
			m, err := spec.AccessMethod(src)
			if err != nil {
				return nil, "", nil, err
			}
			return m.AsBlobAccess(), cpi.ReferenceHint(spec, src), cpi.GlobalAccess(spec, tgtbridge.GetContext()), nil
		}
		return nil, "", nil, nil
	}

	return tgtbridge.(*componentVersionAccessBridge).setupLocalBlobs(kind, prov, it, false, opts)
}
