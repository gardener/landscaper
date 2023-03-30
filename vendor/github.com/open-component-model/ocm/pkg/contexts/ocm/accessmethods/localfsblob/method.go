// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package localfsblob

import (
	"fmt"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/runtime"
)

// Type is the access type of a blob in a local filesystem.
const (
	Type   = "localFilesystemBlob"
	TypeV1 = Type + runtime.VersionSeparator + "v1"
)

// Keep old access method and map generic one to this implementation for component archives

func init() {
	cpi.RegisterAccessType(cpi.NewConvertedAccessSpecType(Type, LocalFilesystemBlobV1))
	cpi.RegisterAccessType(cpi.NewConvertedAccessSpecType(TypeV1, LocalFilesystemBlobV1))
}

// New creates a new localFilesystemBlob accessor.
func New(path string, media string) *localblob.AccessSpec {
	return &localblob.AccessSpec{
		ObjectVersionedType: runtime.NewVersionedObjectType(Type),
		LocalReference:      path,
		MediaType:           media,
	}
}

// AccessSpec describes the access for a blob on the filesystem.
// Deprecated: use LocalBlob.
type AccessSpec struct {
	runtime.ObjectVersionedType `json:",inline"`
	// FileName is the
	Filename string `json:"fileName"`
	// MediaType is the media type of the object represented by the blob
	MediaType string `json:"mediaType"`
}

////////////////////////////////////////////////////////////////////////////////

type localfsblobConverterV1 struct{}

var LocalFilesystemBlobV1 = cpi.NewAccessSpecVersion(&AccessSpec{}, localfsblobConverterV1{})

func (_ localfsblobConverterV1) ConvertFrom(object cpi.AccessSpec) (runtime.TypedObject, error) {
	in, ok := object.(*localblob.AccessSpec)
	if !ok {
		return nil, fmt.Errorf("failed to assert type %T to localblob.AccessSpec", object)
	}
	return &AccessSpec{
		ObjectVersionedType: runtime.NewVersionedObjectType(in.Type),
		Filename:            in.LocalReference,
		MediaType:           in.MediaType,
	}, nil
}

func (_ localfsblobConverterV1) ConvertTo(object interface{}) (cpi.AccessSpec, error) {
	in, ok := object.(*AccessSpec)
	if !ok {
		return nil, fmt.Errorf("failed to assert type %T to localfsblob.AccessSpec", object)
	}
	return &localblob.AccessSpec{
		ObjectVersionedType: runtime.NewVersionedObjectType(in.Type),
		LocalReference:      in.Filename,
		MediaType:           in.MediaType,
	}, nil
}
