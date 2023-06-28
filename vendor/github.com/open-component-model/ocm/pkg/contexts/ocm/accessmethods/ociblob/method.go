// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociblob

import (
	"fmt"
	"io"
	"sync"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/ocireg"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
)

// Type is the access type for a blob in an OCI repository.
const (
	Type   = "ociBlob"
	TypeV1 = Type + runtime.VersionSeparator + "v1"
)

func init() {
	cpi.RegisterAccessType(cpi.NewAccessSpecType[*AccessSpec](Type, cpi.WithDescription(usage)))
	cpi.RegisterAccessType(cpi.NewAccessSpecType[*AccessSpec](TypeV1, cpi.WithFormatSpec(formatV1), cpi.WithConfigHandler(ConfigHandler())))
}

// New creates a new OCIBlob accessor.
func New(repository string, digest digest.Digest, mediaType string, size int64) *AccessSpec {
	return &AccessSpec{
		ObjectVersionedType: runtime.NewVersionedTypedObject(Type),
		Reference:           repository,
		MediaType:           mediaType,
		Digest:              digest,
		Size:                size,
	}
}

// AccessSpec describes the access for a oci registry.
type AccessSpec struct {
	runtime.ObjectVersionedType `json:",inline"`

	// Reference is the oci reference to the OCI repository
	Reference string `json:"ref"`

	// MediaType is the media type of the object this schema refers to.
	MediaType string `json:"mediaType,omitempty"`

	// Digest is the digest of the targeted content.
	Digest digest.Digest `json:"digest"`

	// Size specifies the size in bytes of the blob.
	Size int64 `json:"size"`
}

var _ cpi.AccessSpec = (*AccessSpec)(nil)

func (a *AccessSpec) Describe(ctx cpi.Context) string {
	return fmt.Sprintf("OCI blob %s in repository %s", a.Digest, a.Reference)
}

func (s *AccessSpec) IsLocal(context cpi.Context) bool {
	return false
}

func (s *AccessSpec) GlobalAccessSpec(ctx cpi.Context) cpi.AccessSpec {
	return s
}

func (s *AccessSpec) GetMimeType() string {
	return s.MediaType
}

func (s *AccessSpec) AccessMethod(access cpi.ComponentVersionAccess) (cpi.AccessMethod, error) {
	return &accessMethod{comp: access, spec: s}, nil
}

func (s *AccessSpec) GetInexpensiveContentVersionIdentity(access cpi.ComponentVersionAccess) string {
	return s.Digest.String()
}

////////////////////////////////////////////////////////////////////////////////

// TODO add cache

type accessMethod struct {
	lock sync.Mutex
	blob accessio.BlobAccess
	comp cpi.ComponentVersionAccess
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
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.blob != nil {
		m.blob.Close()
		m.blob = nil
	}
	return nil
}

func (m *accessMethod) Get() ([]byte, error) {
	return accessio.BlobData(m.getBlob())
}

func (m *accessMethod) Reader() (io.ReadCloser, error) {
	return accessio.BlobReader(m.getBlob())
}

func (m *accessMethod) MimeType() string {
	return m.spec.MediaType
}

func (m *accessMethod) getBlob() (cpi.BlobAccess, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.blob != nil {
		return m.blob, nil
	}
	ref, err := oci.ParseRef(m.spec.Reference)
	if err != nil {
		return nil, err
	}
	if ref.Tag != nil || ref.Digest != nil {
		return nil, errors.ErrInvalid("oci repository", m.spec.Reference)
	}
	ocictx := m.comp.GetContext().OCIContext()
	spec := ocictx.GetAlias(ref.Host)
	if spec == nil {
		spec = ocireg.NewRepositorySpec(ref.Host)
	}
	ocirepo, err := m.comp.GetContext().OCIContext().RepositoryForSpec(spec)
	if err != nil {
		return nil, err
	}
	ns, err := ocirepo.LookupNamespace(ref.Repository)
	if err != nil {
		return nil, err
	}
	size, acc, err := ns.GetBlobData(m.spec.Digest)
	if err != nil {
		return nil, err
	}
	if m.spec.Size == accessio.BLOB_UNKNOWN_SIZE {
		m.spec.Size = size
	} else if size != accessio.BLOB_UNKNOWN_SIZE {
		return nil, errors.Newf("blob size mismatch %d != %d", size, m.spec.Size)
	}
	m.blob = accessio.BlobAccessForDataAccess(m.spec.Digest, m.spec.Size, m.spec.MediaType, acc)
	return m.blob, nil
}
