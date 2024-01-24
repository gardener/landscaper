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

const TypeV1 = Type + runtime.VersionSeparator + "v1"

func initV1() {
	Must(versions.Register(accspeccpi.NewAccessSpecTypeByConverter[*AccessSpec, *AccessSpecV1](TypeV1, &converterV1{}, accspeccpi.WithFormatSpec(formatV1))))
	Must(versions.Register(accspeccpi.NewAccessSpecTypeByConverter[*AccessSpec, *AccessSpecV1](LegacyTypeV1, &converterV1{}, accspeccpi.WithFormatSpec(formatV1))))

	formats.Register(TypeV1, runtime.NewConvertedVersion[cpi.AccessSpec, *AccessSpec, *AccessSpecV1](&converterV1{}))
	formats.Register(LegacyTypeV1, runtime.NewConvertedVersion[cpi.AccessSpec, *AccessSpec, *AccessSpecV1](&converterV1{}))
}

// AccessSpecV1 describes the v1 format.
type AccessSpecV1 struct {
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
	MediaType string `json:"mediaType,omitempty"`
}

type converterV1 struct{}

func (_ converterV1) ConvertFrom(in *AccessSpec) (*AccessSpecV1, error) {
	return &AccessSpecV1{
		ObjectVersionedType: runtime.NewVersionedTypedObject(in.Type),
		Region:              in.Region,
		Bucket:              in.Bucket,
		Key:                 in.Key,
		Version:             in.Version,
		MediaType:           in.MediaType,
	}, nil
}

func (_ converterV1) ConvertTo(in *AccessSpecV1) (*AccessSpec, error) {
	return &AccessSpec{
		InternalVersionedTypedObject: runtime.NewInternalVersionedTypedObject[cpi.AccessSpec](versions, in.Type),
		Region:                       in.Region,
		Bucket:                       in.Bucket,
		Key:                          in.Key,
		Version:                      in.Version,
		MediaType:                    in.MediaType,
	}, nil
}

var formatV1 = `
The type specific specification fields are:

- **<code>region</code>** (optional) *string*

  OCI repository reference (this artifact name used to store the blob).

- **<code>bucket</code>** *string*

  The name of the S3 bucket containing the blob

- **<code>key</code>** *string*

  The key of the desired blob

- **<code>version</code>** (optional) *string*

  The key of the desired blob

- **<code>mediaType</code>** (optional) *string*

  The media type of the content
`
