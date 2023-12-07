// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package s3

import (
	. "github.com/open-component-model/ocm/pkg/exception"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/accspeccpi"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const TypeV2 = Type + runtime.VersionSeparator + "v2"

const LegacyTypeV2 = LegacyType + runtime.VersionSeparator + "v2"

func initV2() {
	Must(versions.Register(accspeccpi.NewAccessSpecTypeByConverter[*AccessSpec, *AccessSpecV2](TypeV2, &converterV2{}, accspeccpi.WithFormatSpec(formatV2))))
	Must(versions.Register(accspeccpi.NewAccessSpecTypeByConverter[*AccessSpec, *AccessSpecV2](LegacyTypeV2, &converterV2{}, accspeccpi.WithFormatSpec(formatV2))))

	formats.Register(TypeV2, runtime.NewConvertedVersion[cpi.AccessSpec, *AccessSpec, *AccessSpecV2](&converterV2{}))
	formats.Register(LegacyTypeV2, runtime.NewConvertedVersion[cpi.AccessSpec, *AccessSpec, *AccessSpecV2](&converterV2{}))
}

// AccessSpecV2 describes the v2 format.
type AccessSpecV2 struct {
	runtime.ObjectVersionedType `json:",inline"`

	// Region needs to be set even though buckets are global.
	// We can't assume that there is a default region setting sitting somewhere.
	// +optional
	Region string `json:"region,omitempty"`
	// Bucket where the s3 object is located.
	Bucket string `json:"bucketName"`
	// Key of the object to look for. This value will be used together with Bucket and Version to form an identity.
	Key string `json:"objectKey"`
	// Version of the object.
	// +optional
	Version string `json:"version,omitempty"`
	// MediaType defines the mime type of the object to download.
	// +optional
	MediaType string `json:"mediaType,omitempty"`
}

type converterV2 struct{}

func (_ converterV2) ConvertFrom(in *AccessSpec) (*AccessSpecV2, error) {
	return &AccessSpecV2{
		ObjectVersionedType: runtime.NewVersionedTypedObject(in.Type),
		Region:              in.Region,
		Bucket:              in.Bucket,
		Key:                 in.Key,
		Version:             in.Version,
		MediaType:           in.MediaType,
	}, nil
}

func (_ converterV2) ConvertTo(in *AccessSpecV2) (*AccessSpec, error) {
	return &AccessSpec{
		InternalVersionedTypedObject: runtime.NewInternalVersionedTypedObject[cpi.AccessSpec](versions, in.Type),
		Region:                       in.Region,
		Bucket:                       in.Bucket,
		Key:                          in.Key,
		Version:                      in.Version,
		MediaType:                    in.MediaType,
	}, nil
}

var formatV2 = `
The type specific specification fields are:

- **<code>region</code>** (optional) *string*

  OCI repository reference (this artifact name used to store the blob).

- **<code>bucketName</code>** *string*

  The name of the S3 bucket containing the blob

- **<code>objectKey</code>** *string*

  The key of the desired blob

- **<code>version</code>** (optional) *string*

  The key of the desired blob

- **<code>mediaType</code>** (optional) *string*

  The media type of the content
`
