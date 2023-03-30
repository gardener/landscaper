// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package genericocireg

import (
	"io"
	"sync"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localociblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
)

type localOCIBlobAccessMethod struct {
	lock   sync.Mutex
	data   accessio.DataAccess
	spec   *localociblob.AccessSpec
	access oci.NamespaceAccess
}

var _ cpi.AccessMethod = (*localOCIBlobAccessMethod)(nil)

func newLocalOCIBlobAccessMethod(a *localociblob.AccessSpec, access oci.NamespaceAccess) (cpi.AccessMethod, error) {
	return &localOCIBlobAccessMethod{
		spec:   a,
		access: access,
	}, nil
}

func (m *localOCIBlobAccessMethod) GetKind() string {
	return localociblob.Type
}

func (m *localOCIBlobAccessMethod) AccessSpec() cpi.AccessSpec {
	return m.spec
}

func (m *localOCIBlobAccessMethod) Close() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.data != nil {
		tmp := m.data
		m.data = nil
		return tmp.Close()
	}
	return nil
}

func (m *localOCIBlobAccessMethod) getBlob() (cpi.DataAccess, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.data != nil {
		return m.data, nil
	}
	_, data, err := m.access.GetBlobData(m.spec.Digest)
	if err != nil {
		return nil, err
	}
	m.data = data
	return m.data, err
}

func (m *localOCIBlobAccessMethod) Reader() (io.ReadCloser, error) {
	blob, err := m.getBlob()
	if err != nil {
		return nil, err
	}
	return blob.Reader()
}

func (m *localOCIBlobAccessMethod) Get() ([]byte, error) {
	return accessio.BlobData(m.getBlob())
}

func (m *localOCIBlobAccessMethod) MimeType() string {
	return m.spec.GetMimeType()
}
