// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package none

import (
	"io"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
)

// Type is the access type for no blob.
const (
	Type       = compdesc.NoneType
	TypeV1     = Type + runtime.VersionSeparator + "v1"
	LegacyType = compdesc.NoneLegacyType
)

func init() {
	cpi.RegisterAccessType(cpi.NewAccessSpecType[*AccessSpec](Type, cpi.WithDescription("dummy resource with no access")))
	cpi.RegisterAccessType(cpi.NewAccessSpecType[*AccessSpec](TypeV1))
	cpi.RegisterAccessType(cpi.NewAccessSpecType[*AccessSpec](LegacyType))
}

// New creates a new OCIBlob accessor.
func New() *AccessSpec {
	return &AccessSpec{ObjectVersionedType: runtime.NewVersionedTypedObject(Type)}
}

func IsNone(kind string) bool {
	return compdesc.IsNoneAccessKind(kind)
}

// AccessSpec describes the access for a oci registry.
type AccessSpec struct {
	runtime.ObjectVersionedType `json:",inline"`
}

var _ cpi.AccessSpec = (*AccessSpec)(nil)

func (a *AccessSpec) Describe(ctx cpi.Context) string {
	return "none"
}

func (s *AccessSpec) IsLocal(context cpi.Context) bool {
	return false
}

func (s *AccessSpec) GlobalAccessSpec(ctx cpi.Context) cpi.AccessSpec {
	return nil
}

func (s *AccessSpec) GetMimeType() string {
	return ""
}

func (s *AccessSpec) AccessMethod(access cpi.ComponentVersionAccess) (cpi.AccessMethod, error) {
	return &accessMethod{spec: s}, nil
}

func (s *AccessSpec) GetInexpensiveContentVersionIdentity(access cpi.ComponentVersionAccess) string {
	return ""
}

////////////////////////////////////////////////////////////////////////////////

type accessMethod struct {
	spec *AccessSpec
}

var _ cpi.AccessMethod = (*accessMethod)(nil)

func (m *accessMethod) GetKind() string {
	return Type
}

func (m *accessMethod) AccessSpec() cpi.AccessSpec {
	return m.spec
}

func (m *accessMethod) Close() error {
	return nil
}

func (m *accessMethod) Get() ([]byte, error) {
	return nil, errors.ErrNotSupported("access")
}

func (m *accessMethod) Reader() (io.ReadCloser, error) {
	return nil, errors.ErrNotSupported("access")
}

func (m *accessMethod) MimeType() string {
	return ""
}
