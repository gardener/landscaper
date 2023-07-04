// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package comparch

import (
	"io"
	"sync"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/support"
)

////////////////////////////////////////////////////////////////////////////////

type localFilesystemBlobAccessMethod struct {
	sync.RWMutex
	spec       *localblob.AccessSpec
	base       support.ComponentVersionContainer
	blobAccess accessio.DataAccess
}

var _ cpi.AccessMethod = (*localFilesystemBlobAccessMethod)(nil)

func newLocalFilesystemBlobAccessMethod(a *localblob.AccessSpec, base support.ComponentVersionContainer) cpi.AccessMethod {
	return &localFilesystemBlobAccessMethod{
		spec: a,
		base: base,
	}
}

func (m *localFilesystemBlobAccessMethod) AccessSpec() cpi.AccessSpec {
	return m.spec
}

func (m *localFilesystemBlobAccessMethod) GetKind() string {
	return localblob.Type
}

func (m *localFilesystemBlobAccessMethod) Reader() (io.ReadCloser, error) {
	m.Lock()
	defer m.Unlock()

	if m.blobAccess == nil {
		var err error
		m.blobAccess, err = m.base.GetBlobData(m.spec.LocalReference)
		if err != nil {
			return accessio.BlobReader(m.blobAccess, err)
		}
	}
	return accessio.BlobReader(m.blobAccess, nil)
}

func (m *localFilesystemBlobAccessMethod) Get() ([]byte, error) {
	m.Lock()
	defer m.Unlock()

	if m.blobAccess == nil {
		var err error
		m.blobAccess, err = m.base.GetBlobData(m.spec.LocalReference)
		if err != nil {
			return accessio.BlobData(m.blobAccess, err)
		}
	}
	return accessio.BlobData(m.blobAccess, nil)
}

func (m *localFilesystemBlobAccessMethod) MimeType() string {
	return m.spec.MediaType
}

func (m *localFilesystemBlobAccessMethod) Close() error {
	m.Lock()
	defer m.Unlock()

	if m.blobAccess != nil {
		return m.blobAccess.Close()
	}
	return nil
}
