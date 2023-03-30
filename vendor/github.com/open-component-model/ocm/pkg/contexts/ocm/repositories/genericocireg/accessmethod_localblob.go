// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package genericocireg

import (
	"io"
	"sync"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
)

type localBlobAccessMethod struct {
	lock   sync.Mutex
	data   accessio.DataAccess
	ctx    cpi.Context
	spec   *localblob.AccessSpec
	access oci.NamespaceAccess
}

var _ cpi.AccessMethod = (*localBlobAccessMethod)(nil)

func newLocalBlobAccessMethod(a *localblob.AccessSpec, access oci.NamespaceAccess, ctx cpi.Context) (cpi.AccessMethod, error) {
	return &localBlobAccessMethod{
		ctx:    ctx,
		spec:   a,
		access: access,
	}, nil
}

func (m *localBlobAccessMethod) GetKind() string {
	return localblob.Type
}

func (m *localBlobAccessMethod) AccessSpec() cpi.AccessSpec {
	return m.spec
}

func (m *localBlobAccessMethod) Close() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.data != nil {
		tmp := m.data
		m.data = nil
		return tmp.Close()
	}
	return nil
}

func (m *localBlobAccessMethod) getBlob() (cpi.DataAccess, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.data != nil {
		return m.data, nil
	}
	if artdesc.IsOCIMediaType(m.spec.MediaType) {
		// may be we should always store the blob, additionally to the
		// exploded form to make things easier.

		if m.spec.LocalReference == "" {
			// TODO: synthesize the artifact blob
			return nil, errors.ErrNotImplemented("artifact blob synthesis")
		}
	}
	_, data, err := m.access.GetBlobData(digest.Digest(m.spec.LocalReference))
	if err != nil {
		return nil, err
	}
	m.data = data
	return m.data, err
}

func (m *localBlobAccessMethod) Reader() (io.ReadCloser, error) {
	blob, err := m.getBlob()
	if err != nil {
		return nil, err
	}
	return blob.Reader()
}

func (m *localBlobAccessMethod) Get() ([]byte, error) {
	return accessio.BlobData(m.getBlob())
}

func (m *localBlobAccessMethod) MimeType() string {
	return m.spec.MediaType
}
