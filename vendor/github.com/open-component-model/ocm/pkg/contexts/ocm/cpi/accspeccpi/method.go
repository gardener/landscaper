// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accspeccpi

import (
	"io"
	"sync"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/utils"
)

////////////////////////////////////////////////////////////////////////////////

type DefaultAccessMethodImpl struct {
	lock sync.Mutex
	blob blobaccess.BlobAccess

	factory BlobAccessFactory
	comp    ComponentVersionAccess
	spec    AccessSpec
	mime    string
	digest  digest.Digest
	local   bool
}

var (
	_ AccessMethodImpl        = (*DefaultAccessMethodImpl)(nil)
	_ blobaccess.DigestSource = (*DefaultAccessMethodImpl)(nil)
)

type BlobAccessFactory func() (blobaccess.BlobAccess, error)

func NewDefaultMethod(c ComponentVersionAccess, a AccessSpec, digest digest.Digest, mime string, fac BlobAccessFactory, local ...bool) AccessMethod {
	m, _ := AccessMethodForImplementation(NewDefaultMethodImpl(c, a, digest, mime, fac, local...), nil)
	return m
}

func NewDefaultMethodImpl(c ComponentVersionAccess, a AccessSpec, digest digest.Digest, mime string, fac BlobAccessFactory, local ...bool) AccessMethodImpl {
	return &DefaultAccessMethodImpl{
		spec:    a,
		comp:    c,
		mime:    mime,
		digest:  digest,
		factory: fac,
		local:   utils.Optional(local...),
	}
}

func NewDefaultMethodForBlobAccess(c ComponentVersionAccess, a AccessSpec, digest digest.Digest, blob blobaccess.BlobAccess, local ...bool) (AccessMethod, error) {
	return AccessMethodForImplementation(NewDefaultMethodImplForBlobAccess(c, a, digest, blob, local...))
}

func NewDefaultMethodImplForBlobAccess(c ComponentVersionAccess, a AccessSpec, digest digest.Digest, blob blobaccess.BlobAccess, local ...bool) (AccessMethodImpl, error) {
	blob, err := blob.Dup()
	if err != nil {
		return nil, err
	}
	return &DefaultAccessMethodImpl{
		spec:    a,
		blob:    blob,
		comp:    c,
		mime:    blob.MimeType(),
		digest:  digest,
		factory: nil,
		local:   utils.Optional(local...),
	}, nil
}

func (m *DefaultAccessMethodImpl) getAccess() (blobaccess.BlobAccess, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.blob == nil {
		acc, err := m.factory()
		if err != nil {
			return nil, err
		}
		m.blob = acc
	}
	return m.blob, nil
}

func (m *DefaultAccessMethodImpl) Digest() digest.Digest {
	return m.digest
}

func (m *DefaultAccessMethodImpl) IsLocal() bool {
	return m.local
}

func (m *DefaultAccessMethodImpl) GetKind() string {
	return m.spec.GetKind()
}

func (m *DefaultAccessMethodImpl) AccessSpec() AccessSpec {
	return m.spec
}

func (m *DefaultAccessMethodImpl) Get() ([]byte, error) {
	a, err := m.getAccess()
	if err != nil {
		return nil, err
	}
	return a.Get()
}

func (m *DefaultAccessMethodImpl) Reader() (io.ReadCloser, error) {
	a, err := m.getAccess()
	if err != nil {
		return nil, err
	}
	return a.Reader()
}

func (m *DefaultAccessMethodImpl) Close() error {
	var err error
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.blob != nil {
		err = m.blob.Close()
		m.blob = nil
	}
	return err
}

func (m *DefaultAccessMethodImpl) MimeType() string {
	return m.mime
}
