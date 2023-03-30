// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package s3

import (
	"fmt"
	"path"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessio/downloader"
	"github.com/open-component-model/ocm/pkg/common/accessio/downloader/s3"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/identity/hostpath"
	"github.com/open-component-model/ocm/pkg/contexts/oci/identity"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/mime"
	"github.com/open-component-model/ocm/pkg/runtime"
)

// Type is the access type of S3 registry.
const (
	Type   = "s3"
	TypeV1 = Type + runtime.VersionSeparator + "v1"

	LegacyType    = "S3"
	LegacyTypeV1  = LegacyType + runtime.VersionSeparator + "v1"
	CONSUMER_TYPE = "S3"
)

func init() {
	cpi.RegisterAccessType(cpi.NewAccessSpecType(Type, &AccessSpec{}, cpi.WithDescription(usage)))
	cpi.RegisterAccessType(cpi.NewAccessSpecType(TypeV1, &AccessSpec{}, cpi.WithFormatSpec(formatV1), cpi.WithConfigHandler(ConfigHandler())))
	cpi.RegisterAccessType(cpi.NewAccessSpecType(LegacyType, &AccessSpec{}))
	cpi.RegisterAccessType(cpi.NewAccessSpecType(LegacyTypeV1, &AccessSpec{}))
}

// AccessSpec describes the access for a GitHub registry.
type AccessSpec struct {
	runtime.ObjectVersionedType `json:",inline"`

	// Region needs to be set even though buckets are global.
	// We can't assume that there is a default region setting sitting somewhere.
	// +optional
	Region string `json:"region,omitempty"`
	// Bucket where the s3 object is located.
	Bucket string `json:"bucket"`
	// Key of the object to look for. This value will be used together with Bucket and Version to form an identity.
	Key string `json:"key"`
	// Version of the object.
	// +optional
	Version string `json:"version,omitempty"`
	// MediaType defines the mime type of the object to download.
	// +optional
	MediaType  string `json:"mediaType,omitempty"`
	downloader downloader.Downloader
}

var _ cpi.AccessSpec = (*AccessSpec)(nil)

// New creates a new GitHub registry access spec version v1.
func New(region, bucket, key, version, mediaType string, downloader downloader.Downloader) *AccessSpec {
	return &AccessSpec{
		ObjectVersionedType: runtime.NewVersionedObjectType(Type),
		Region:              region,
		Bucket:              bucket,
		Key:                 key,
		Version:             version,
		MediaType:           mediaType,
		downloader:          downloader,
	}
}

func (a *AccessSpec) Describe(ctx cpi.Context) string {
	return fmt.Sprintf("S3 key %s in bucket %s", a.Key, a.Bucket)
}

func (_ *AccessSpec) IsLocal(cpi.Context) bool {
	return false
}

func (_ *AccessSpec) GetType() string {
	return Type
}

func (a *AccessSpec) AccessMethod(c cpi.ComponentVersionAccess) (cpi.AccessMethod, error) {
	return newMethod(c, a)
}

////////////////////////////////////////////////////////////////////////////////

type accessMethod struct {
	accessio.BlobAccess

	comp cpi.ComponentVersionAccess
	spec *AccessSpec
}

var _ cpi.AccessMethod = (*accessMethod)(nil)

func newMethod(c cpi.ComponentVersionAccess, a *AccessSpec) (*accessMethod, error) {
	creds, err := getCreds(a, c.GetContext().CredentialsContext())
	if err != nil {
		return nil, fmt.Errorf("failed to get creds: %w", err)
	}

	var (
		accessKeyID  string
		accessSecret string
	)
	if creds != nil {
		accessKeyID = creds.GetProperty(credentials.ATTR_AWS_ACCESS_KEY_ID)
		accessSecret = creds.GetProperty(credentials.ATTR_AWS_SECRET_ACCESS_KEY)
	}
	var awsCreds *s3.AWSCreds
	if accessKeyID != "" {
		awsCreds = &s3.AWSCreds{
			AccessKeyID:  accessKeyID,
			AccessSecret: accessSecret,
		}
	}
	var d downloader.Downloader = s3.NewDownloader(a.Region, a.Bucket, a.Key, a.Version, awsCreds)
	if a.downloader != nil {
		d = a.downloader
	}
	w := accessio.NewWriteAtWriter(d.Download)
	// don't change the spec, leave it empty.
	mediaType := a.MediaType
	if mediaType == "" {
		mediaType = mime.MIME_OCTET
	}
	cacheBlobAccess := accessobj.CachedBlobAccessForWriter(c.GetContext(), mediaType, w)
	return &accessMethod{
		spec:       a,
		comp:       c,
		BlobAccess: cacheBlobAccess,
	}, nil
}

func getCreds(a *AccessSpec, cctx credentials.Context) (credentials.Credentials, error) {
	id := credentials.ConsumerIdentity{
		identity.ID_TYPE:     CONSUMER_TYPE,
		identity.ID_HOSTNAME: a.Bucket,
	}
	if a.Version != "" {
		id[identity.ID_PORT] = a.Version
	}
	id[identity.ID_PATHPREFIX] = path.Join(a.Bucket, a.Key, a.Version)
	return credentials.CredentialsForConsumer(cctx,
		credentials.ConsumerIdentity{
			identity.ID_TYPE: CONSUMER_TYPE,
		},
		hostpath.IdentityMatcher(CONSUMER_TYPE))
}

func (m *accessMethod) GetKind() string {
	return Type
}

func (m *accessMethod) AccessSpec() cpi.AccessSpec {
	return m.spec
}
