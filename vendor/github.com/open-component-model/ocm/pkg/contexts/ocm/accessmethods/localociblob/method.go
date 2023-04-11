// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package localociblob

import (
	"fmt"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/runtime"
)

// Type is the access type for a component version local blob in an OCI repository.
const (
	Type   = "localOciBlob"
	TypeV1 = Type + runtime.VersionSeparator + "v1"
)

func init() {
	cpi.RegisterAccessType(cpi.NewAccessSpecType(Type, &AccessSpec{}))
	cpi.RegisterAccessType(cpi.NewAccessSpecType(TypeV1, &AccessSpec{}))
}

// New creates a new LocalOCIBlob accessor.
// Deprecated: Use LocalBlob.
func New(digest digest.Digest) *AccessSpec {
	return &AccessSpec{
		ObjectVersionedType: runtime.NewVersionedObjectType(Type),
		Digest:              digest,
	}
}

// AccessSpec describes the access for a oci registry.
// Deprecated: Use LocalBlob.
type AccessSpec struct {
	runtime.ObjectVersionedType `json:",inline"`

	// Digest is the digest of the targeted content.
	Digest digest.Digest `json:"digest"`
}

var _ cpi.AccessSpec = (*AccessSpec)(nil)

func (a *AccessSpec) Describe(ctx cpi.Context) string {
	return fmt.Sprintf("Local OCI blob %s", a.Digest)
}

func (s AccessSpec) IsLocal(context cpi.Context) bool {
	return true
}

func (a *AccessSpec) GlobalAccessSpec(ctx cpi.Context) cpi.AccessSpec {
	return nil
}

func (s *AccessSpec) GetMimeType() string {
	return ""
}

func (s *AccessSpec) AccessMethod(c cpi.ComponentVersionAccess) (cpi.AccessMethod, error) {
	return c.AccessMethod(s)
}
