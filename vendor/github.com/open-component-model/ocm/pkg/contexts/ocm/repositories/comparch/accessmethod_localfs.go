// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package comparch

import (
	"io"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/support"
)

////////////////////////////////////////////////////////////////////////////////

type localFilesystemBlobAccessMethod struct {
	accessio.NopCloser
	spec *localblob.AccessSpec
	base support.ComponentVersionContainer
}

var _ cpi.AccessMethod = (*localFilesystemBlobAccessMethod)(nil)

func newLocalFilesystemBlobAccessMethod(a *localblob.AccessSpec, base support.ComponentVersionContainer) (cpi.AccessMethod, error) {
	return &localFilesystemBlobAccessMethod{
		spec: a,
		base: base,
	}, nil
}

func (m *localFilesystemBlobAccessMethod) AccessSpec() cpi.AccessSpec {
	return m.spec
}

func (m *localFilesystemBlobAccessMethod) GetKind() string {
	return localblob.Type
}

func (m *localFilesystemBlobAccessMethod) Reader() (io.ReadCloser, error) {
	return accessio.BlobReader(m.base.GetBlobData(m.spec.LocalReference))
}

func (m *localFilesystemBlobAccessMethod) Get() ([]byte, error) {
	return accessio.BlobData(m.base.GetBlobData(m.spec.LocalReference))
}

func (m *localFilesystemBlobAccessMethod) MimeType() string {
	return m.spec.MediaType
}
