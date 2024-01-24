// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"fmt"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	ocm "github.com/open-component-model/ocm/pkg/contexts/ocm/context"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/accspeccpi"
	cpi "github.com/open-component-model/ocm/pkg/contexts/ocm/internal"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/plugin/descriptor"
	"github.com/open-component-model/ocm/pkg/errors"
)

////////////////////////////////////////////////////////////////////////////////

// ComponentVersionProvider should be implemented
// by Accesses based on component version instances.
// It is used to determine access type specific
// information. For example, OCI based access types
// may provide global OCI artifact references.
type ComponentVersionProvider interface {
	GetComponentVersion() (ComponentVersionAccess, error)
}

type ComponentVersionBasedAccessProvider struct {
	vers   ComponentVersionAccess
	access compdesc.AccessSpec
}

var (
	_ AccessProvider           = (*ComponentVersionBasedAccessProvider)(nil)
	_ ComponentVersionProvider = (*ComponentVersionBasedAccessProvider)(nil)
)

// Deprecated: use ComponentVersionBasedAccessProvider.
type BaseAccess = ComponentVersionBasedAccessProvider

func NewBaseAccess(cv ComponentVersionAccess, acc compdesc.AccessSpec) *ComponentVersionBasedAccessProvider {
	return &ComponentVersionBasedAccessProvider{vers: cv, access: acc}
}

func (r *ComponentVersionBasedAccessProvider) GetOCMContext() Context {
	return r.vers.GetContext()
}

func (r *ComponentVersionBasedAccessProvider) GetComponentVersion() (ComponentVersionAccess, error) {
	return r.vers.Dup()
}

func (r *ComponentVersionBasedAccessProvider) ReferenceHint() string {
	if hp, ok := r.access.(cpi.HintProvider); ok {
		return hp.GetReferenceHint(r.vers)
	}
	return ""
}

func (r *ComponentVersionBasedAccessProvider) GlobalAccess() AccessSpec {
	acc, err := r.GetOCMContext().AccessSpecForSpec(r.access)
	if err != nil {
		return nil
	}
	return acc.GlobalAccessSpec(r.GetOCMContext())
}

func (r *ComponentVersionBasedAccessProvider) Access() (AccessSpec, error) {
	return r.vers.GetContext().AccessSpecForSpec(r.access)
}

func (r *ComponentVersionBasedAccessProvider) AccessMethod() (AccessMethod, error) {
	acc, err := r.vers.GetContext().AccessSpecForSpec(r.access)
	if err != nil {
		return nil, err
	}
	return acc.AccessMethod(r.vers)
}

func (r *ComponentVersionBasedAccessProvider) BlobAccess() (BlobAccess, error) {
	m, err := r.AccessMethod()
	if err != nil {
		return nil, err
	}
	return m.AsBlobAccess(), nil
}

////////////////////////////////////////////////////////////////////////////////

type blobAccessProvider struct {
	ctx ocm.Context
	blobaccess.BlobAccessProvider
	hint   string
	global AccessSpec
}

var _ AccessProvider = (*blobAccessProvider)(nil)

func NewAccessProviderForBlobAccessProvider(ctx ocm.Context, prov blobaccess.BlobAccessProvider, hint string, global AccessSpec) AccessProvider {
	return &blobAccessProvider{
		BlobAccessProvider: prov,
		hint:               hint,
		global:             global,
		ctx:                ctx,
	}
}

func (b *blobAccessProvider) GetOCMContext() cpi.Context {
	return b.ctx
}

func (b *blobAccessProvider) ReferenceHint() string {
	return b.hint
}

func (b *blobAccessProvider) GlobalAccess() cpi.AccessSpec {
	return b.global
}

func (b blobAccessProvider) Access() (cpi.AccessSpec, error) {
	return nil, errors.ErrNotFound(descriptor.KIND_ACCESSMETHOD)
}

func (b *blobAccessProvider) AccessMethod() (cpi.AccessMethod, error) {
	return nil, errors.ErrNotFound(descriptor.KIND_ACCESSMETHOD)
}

////////////////////////////////////////////////////////////////////////////////

func NewArtifactAccessProviderForBlobAccessProvider[M any](ctx Context, meta *M, src blobAccessProvider, hint string, global AccessSpec) cpi.ArtifactAccess[M] {
	return NewArtifactAccessForProvider(meta, NewAccessProviderForBlobAccessProvider(ctx, src, hint, global))
}

////////////////////////////////////////////////////////////////////////////////

type accessAccessProvider struct {
	ctx  ocm.Context
	spec AccessSpec
}

var _ AccessProvider = (*accessAccessProvider)(nil)

func NewAccessProviderForExternalAccessSpec(ctx ocm.Context, spec AccessSpec) (AccessProvider, error) {
	if spec.IsLocal(ctx) {
		return nil, fmt.Errorf("access spec describes a repository specific local access method")
	}
	return &accessAccessProvider{
		ctx:  ctx,
		spec: spec,
	}, nil
}

func (b *accessAccessProvider) GetOCMContext() cpi.Context {
	return b.ctx
}

func (b *accessAccessProvider) ReferenceHint() string {
	if h, ok := b.spec.(HintProvider); ok {
		return h.GetReferenceHint(&DummyComponentVersionAccess{b.ctx})
	}
	return ""
}

func (b *accessAccessProvider) GlobalAccess() cpi.AccessSpec {
	return nil
}

func (b *accessAccessProvider) Access() (cpi.AccessSpec, error) {
	return b.spec, nil
}

func (b *accessAccessProvider) AccessMethod() (cpi.AccessMethod, error) {
	return b.spec.AccessMethod(&DummyComponentVersionAccess{b.ctx})
}

func (b *accessAccessProvider) BlobAccess() (blobaccess.BlobAccess, error) {
	return accspeccpi.BlobAccessForAccessSpec(b.spec, &DummyComponentVersionAccess{b.ctx})
}

////////////////////////////////////////////////////////////////////////////////

type (
	accessProvider = AccessProvider
)

type artifactAccessProvider[M any] struct {
	accessProvider
	componentVersionProvider ComponentVersionProvider
	meta                     *M
}

var _ credentials.ConsumerIdentityProvider = (*artifactAccessProvider[any])(nil)

func NewArtifactAccessForProvider[M any](meta *M, prov AccessProvider) cpi.ArtifactAccess[M] {
	aa := &artifactAccessProvider[M]{
		accessProvider: prov,
		meta:           meta,
	}
	if p, ok := prov.(ComponentVersionProvider); ok {
		aa.componentVersionProvider = p
	}
	return aa
}

func (r *artifactAccessProvider[M]) Meta() *M {
	return r.meta
}

func (b *artifactAccessProvider[M]) GetConsumerId(uctx ...credentials.UsageContext) credentials.ConsumerIdentity {
	m, err := b.AccessMethod()
	if err != nil {
		return nil
	}
	defer m.Close()
	return credentials.GetProvidedConsumerId(m, uctx...)
}

func (b *artifactAccessProvider[M]) GetIdentityMatcher() string {
	m, err := b.AccessMethod()
	if err != nil {
		return ""
	}
	defer m.Close()
	return credentials.GetProvidedIdentityMatcher(m)
}

func (b *artifactAccessProvider[M]) GetComponentVersion() (ComponentVersionAccess, error) {
	if b.componentVersionProvider != nil {
		return b.componentVersionProvider.GetComponentVersion()
	}
	return nil, nil
}

////////////////////////////////////////////////////////////////////////////////

var _ ResourceAccess = (*artifactAccessProvider[ResourceMeta])(nil)

func NewResourceAccess(componentVersion ComponentVersionAccess, accessSpec compdesc.AccessSpec, meta ResourceMeta) ResourceAccess {
	return NewResourceAccessForProvider(&meta, NewBaseAccess(componentVersion, accessSpec))
}

func NewResourceAccessForProvider(meta *ResourceMeta, prov AccessProvider) ResourceAccess {
	return NewArtifactAccessForProvider(meta, prov)
}

////////////////////////////////////////////////////////////////////////////////

var _ SourceAccess = (*artifactAccessProvider[SourceMeta])(nil)

func NewSourceAccess(componentVersion ComponentVersionAccess, accessSpec compdesc.AccessSpec, meta SourceMeta) SourceAccess {
	return NewSourceAccessForProvider(&meta, NewBaseAccess(componentVersion, accessSpec))
}

func NewSourceAccessForProvider(meta *SourceMeta, prov AccessProvider) SourceAccess {
	return NewArtifactAccessForProvider(meta, prov)
}
