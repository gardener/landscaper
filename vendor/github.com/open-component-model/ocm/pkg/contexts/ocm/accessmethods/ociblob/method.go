// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociblob

import (
	"fmt"
	"io"
	"sync"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	ociidentity "github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/oci/identity"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/ocireg"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/accspeccpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
)

// Type is the access type for a blob in an OCI repository.
const (
	Type   = "ociBlob"
	TypeV1 = Type + runtime.VersionSeparator + "v1"
)

func init() {
	accspeccpi.RegisterAccessType(accspeccpi.NewAccessSpecType[*AccessSpec](Type, accspeccpi.WithDescription(usage)))
	accspeccpi.RegisterAccessType(accspeccpi.NewAccessSpecType[*AccessSpec](TypeV1, accspeccpi.WithFormatSpec(formatV1), accspeccpi.WithConfigHandler(ConfigHandler())))
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

var _ accspeccpi.AccessSpec = (*AccessSpec)(nil)

func (a *AccessSpec) Describe(ctx accspeccpi.Context) string {
	return fmt.Sprintf("OCI blob %s in repository %s", a.Digest, a.Reference)
}

func (s *AccessSpec) IsLocal(context accspeccpi.Context) bool {
	return false
}

func (s *AccessSpec) GlobalAccessSpec(ctx accspeccpi.Context) accspeccpi.AccessSpec {
	return s
}

func (s *AccessSpec) GetMimeType() string {
	return s.MediaType
}

func (s *AccessSpec) AccessMethod(access accspeccpi.ComponentVersionAccess) (accspeccpi.AccessMethod, error) {
	return accspeccpi.AccessMethodForImplementation(&accessMethod{comp: access, spec: s}, nil)
}

func (s *AccessSpec) GetInexpensiveContentVersionIdentity(access accspeccpi.ComponentVersionAccess) string {
	return s.Digest.String()
}

////////////////////////////////////////////////////////////////////////////////

// TODO add cache

type accessMethod struct {
	lock sync.Mutex
	blob blobaccess.BlobAccess
	comp accspeccpi.ComponentVersionAccess
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
	var err error
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.blob != nil {
		err = m.blob.Close()
		m.blob = nil
	}
	return err
}

func (m *accessMethod) Get() ([]byte, error) {
	return blobaccess.BlobData(m.getBlob())
}

func (m *accessMethod) Reader() (io.ReadCloser, error) {
	return blobaccess.BlobReader(m.getBlob())
}

func (m *accessMethod) MimeType() string {
	return m.spec.MediaType
}

func (m *accessMethod) getBlob() (blobaccess.BlobAccess, error) {
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
	if m.spec.Size == blobaccess.BLOB_UNKNOWN_SIZE {
		m.spec.Size = size
	} else if size != blobaccess.BLOB_UNKNOWN_SIZE {
		return nil, errors.Newf("blob size mismatch %d != %d", size, m.spec.Size)
	}
	m.blob = blobaccess.ForDataAccess(m.spec.Digest, m.spec.Size, m.spec.MediaType, acc)
	return m.blob, nil
}

func (m *accessMethod) GetConsumerId(uctx ...credentials.UsageContext) credentials.ConsumerIdentity {
	m.lock.Lock()
	defer m.lock.Unlock()

	ref, err := oci.ParseRef(m.spec.Reference)
	if err != nil {
		return nil
	}

	ocictx := m.comp.GetContext().OCIContext()
	spec := ocictx.GetAlias(ref.Host)
	if spec == nil {
		spec = ocireg.NewRepositorySpec(ref.Host)
	}
	ocirepo, err := m.comp.GetContext().OCIContext().RepositoryForSpec(spec)
	if err != nil {
		return nil
	}
	return credentials.GetProvidedConsumerId(ocirepo, credentials.StringUsageContext(ref.Repository))
}

func (m *accessMethod) GetIdentityMatcher() string {
	return ociidentity.CONSUMER_TYPE
}
