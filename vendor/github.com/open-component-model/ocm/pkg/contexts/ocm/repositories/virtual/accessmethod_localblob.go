// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package virtual

import (
	"io"
	"sync"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/accspeccpi"
	"github.com/open-component-model/ocm/pkg/errors"
)

type localBlobAccessMethod struct {
	lock sync.Mutex
	data blobaccess.DataAccess
	spec *localblob.AccessSpec
}

var _ accspeccpi.AccessMethodImpl = (*localBlobAccessMethod)(nil)

func newLocalBlobAccessMethod(a *localblob.AccessSpec, data blobaccess.DataAccess) (*localBlobAccessMethod, error) {
	return &localBlobAccessMethod{
		spec: a,
		data: data,
	}, nil
}

func (_ *localBlobAccessMethod) IsLocal() bool {
	return true
}

func (m *localBlobAccessMethod) GetKind() string {
	return m.spec.GetKind()
}

func (m *localBlobAccessMethod) AccessSpec() accspeccpi.AccessSpec {
	return m.spec
}

func (m *localBlobAccessMethod) Close() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.data == nil {
		return blobaccess.ErrClosed
	}
	list := errors.ErrorList{}
	list.Add(m.data.Close())
	m.data = nil
	return list.Result()
}

func (m *localBlobAccessMethod) Reader() (io.ReadCloser, error) {
	return m.data.Reader()
}

func (m *localBlobAccessMethod) Get() (data []byte, ferr error) {
	return blobaccess.BlobData(m.data)
}

func (m *localBlobAccessMethod) MimeType() string {
	return m.spec.MediaType
}
