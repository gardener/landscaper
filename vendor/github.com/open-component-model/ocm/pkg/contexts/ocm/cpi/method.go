// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"io"
	"sync"

	"github.com/open-component-model/ocm/pkg/common/accessio"
)

////////////////////////////////////////////////////////////////////////////////

type DefaultAccessMethod struct {
	lock   sync.Mutex
	access accessio.BlobAccess

	factory BlobAccessFactory
	comp    ComponentVersionAccess
	spec    AccessSpec
	mime    string
}

var _ AccessMethod = (*DefaultAccessMethod)(nil)

type BlobAccessFactory func() (BlobAccess, error)

func NewDefaultMethod(c ComponentVersionAccess, a AccessSpec, mime string, fac BlobAccessFactory) AccessMethod {
	return &DefaultAccessMethod{
		spec:    a,
		comp:    c,
		mime:    mime,
		factory: fac,
	}
}

func (m *DefaultAccessMethod) getAccess() (accessio.BlobAccess, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.access == nil {
		acc, err := m.factory()
		if err != nil {
			return nil, err
		}
		m.access = acc
	}
	return m.access, nil
}

func (m *DefaultAccessMethod) GetKind() string {
	return m.spec.GetKind()
}

func (m *DefaultAccessMethod) AccessSpec() AccessSpec {
	return m.spec
}

func (m *DefaultAccessMethod) Get() ([]byte, error) {
	a, err := m.getAccess()
	if err != nil {
		return nil, err
	}
	return a.Get()
}

func (m *DefaultAccessMethod) Reader() (io.ReadCloser, error) {
	a, err := m.getAccess()
	if err != nil {
		return nil, err
	}
	return a.Reader()
}

func (m *DefaultAccessMethod) Close() error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.access != nil {
		return m.access.Close()
	}
	return nil
}

func (m *DefaultAccessMethod) MimeType() string {
	return m.mime
}
