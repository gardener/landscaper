// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package none

import (
	"io"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/accspeccpi"
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
	accspeccpi.RegisterAccessType(accspeccpi.NewAccessSpecType[*AccessSpec](Type, accspeccpi.WithDescription("dummy resource with no access")))
	accspeccpi.RegisterAccessType(accspeccpi.NewAccessSpecType[*AccessSpec](TypeV1))
	accspeccpi.RegisterAccessType(accspeccpi.NewAccessSpecType[*AccessSpec](LegacyType))
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

var _ accspeccpi.AccessSpec = (*AccessSpec)(nil)

func (a *AccessSpec) Describe(ctx accspeccpi.Context) string {
	return "none"
}

func (s *AccessSpec) IsLocal(context accspeccpi.Context) bool {
	return false
}

func (s *AccessSpec) GlobalAccessSpec(ctx accspeccpi.Context) accspeccpi.AccessSpec {
	return nil
}

func (s *AccessSpec) GetMimeType() string {
	return ""
}

func (s *AccessSpec) AccessMethod(access accspeccpi.ComponentVersionAccess) (accspeccpi.AccessMethod, error) {
	return accspeccpi.AccessMethodForImplementation(&accessMethod{spec: s}, nil)
}

func (s *AccessSpec) GetInexpensiveContentVersionIdentity(access accspeccpi.ComponentVersionAccess) string {
	return ""
}

////////////////////////////////////////////////////////////////////////////////

type accessMethod struct {
	spec *AccessSpec
}

var _ accspeccpi.AccessMethodImpl = (*accessMethod)(nil)

func (_ *accessMethod) IsLocal() bool {
	return false
}

func (m *accessMethod) GetKind() string {
	return Type
}

func (m *accessMethod) AccessSpec() accspeccpi.AccessSpec {
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
