// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package localblob

import (
	"encoding/json"
	"fmt"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/internal"
	"github.com/open-component-model/ocm/pkg/runtime"
)

// Type is the access type of a blob local to a component.
const (
	Type   = "localBlob"
	TypeV1 = Type + runtime.VersionSeparator + "v1"
)

func init() {
	cpi.RegisterAccessType(cpi.NewConvertedAccessSpecType(Type, LocalBlobV1, cpi.WithDescription(usage)))
	cpi.RegisterAccessType(cpi.NewConvertedAccessSpecType(TypeV1, LocalBlobV1, cpi.WithFormatSpec(formatV1), cpi.WithConfigHandler(ConfigHandler())))
}

func Is(spec cpi.AccessSpec) bool {
	return spec != nil && spec.GetKind() == Type
}

// New creates a new localFilesystemBlob accessor.
func New(local, hint string, mediaType string, global cpi.AccessSpec) *AccessSpec {
	return &AccessSpec{
		ObjectVersionedType: runtime.NewVersionedObjectType(Type),
		LocalReference:      local,
		ReferenceName:       hint,
		MediaType:           mediaType,
		GlobalAccess:        internal.NewAccessSpecRef(global),
	}
}

// AccessSpec describes the access for a local blob.
type AccessSpec struct {
	runtime.ObjectVersionedType
	// LocalReference is the repository local identity of the blob.
	// it is used by the repository implementation to get access
	// to the blob and if therefore specific to a dedicated repository type.
	LocalReference string `json:"localReference"`
	// MediaType is the media type of the object represented by the blob
	MediaType string `json:"mediaType"`

	// GlobalAccess is an optional field describing a possibility
	// for a global access. If given, it MUST describe a global access method.
	GlobalAccess *cpi.AccessSpecRef `json:"globalAccess,omitempty"`
	// ReferenceName is an optional static name the object should be
	// use in a local repository context. It is use by a repository
	// to optionally determine a globally referencable access according
	// to the OCI distribution spec. The result will be stored
	// by the repository in the field ImageReference.
	// The value is typically an OCI repository name optionally
	// followed by a colon ':' and a tag
	ReferenceName string `json:"referenceName,omitempty"`
}

var (
	_ json.Marshaler   = (*AccessSpec)(nil)
	_ cpi.HintProvider = (*AccessSpec)(nil)
	_ cpi.AccessSpec   = (*AccessSpec)(nil)
)

func (a AccessSpec) MarshalJSON() ([]byte, error) {
	return cpi.MarshalConvertedAccessSpec(cpi.DefaultContext(), &a)
}

func (a *AccessSpec) Describe(ctx cpi.Context) string {
	return fmt.Sprintf("Local blob %s[%s]", a.LocalReference, a.ReferenceName)
}

func (a *AccessSpec) IsLocal(cpi.Context) bool {
	return true
}

func (a *AccessSpec) GetMimeType() string {
	return a.MediaType
}

func (a *AccessSpec) GetReferenceHint(cv cpi.ComponentVersionAccess) string {
	return a.ReferenceName
}

func (a *AccessSpec) AccessMethod(cv cpi.ComponentVersionAccess) (cpi.AccessMethod, error) {
	return cv.AccessMethod(a)
}

////////////////////////////////////////////////////////////////////////////////

type AccessSpecV1 struct {
	runtime.ObjectVersionedType `json:",inline"`
	// LocalReference is the repository local identity of the blob.
	// it is used by the repository implementation to get access
	// to the blob and if therefore specific to a dedicated repository type.
	LocalReference string `json:"localReference"`
	// MediaType is the media type of the object represented by the blob
	MediaType string `json:"mediaType"`

	// GlobalAccess is an optional field describing a possibility
	// for a global access. If given, it MUST describe a global access method.
	GlobalAccess *internal.AccessSpecRef `json:"globalAccess,omitempty"`
	// ReferenceName is an optional static name the object should be
	// use in a local repository context. It is use by a repository
	// to optionally determine a globally referencable access according
	// to the OCI distribution spec. The result will be stored
	// by the repository in the field ImageReference.
	// The value is typically an OCI repository name optionally
	// followed by a colon ':' and a tag
	ReferenceName string `json:"referenceName,omitempty"`
}

type converterV1 struct{}

var LocalBlobV1 = cpi.NewAccessSpecVersion(&AccessSpecV1{}, converterV1{})

func (_ converterV1) ConvertFrom(object cpi.AccessSpec) (runtime.TypedObject, error) {
	in, ok := object.(*AccessSpec)
	if !ok {
		return nil, fmt.Errorf("failed to assert type %T to AccessSpec", object)
	}
	return &AccessSpecV1{
		ObjectVersionedType: runtime.NewVersionedObjectType(in.Type),
		LocalReference:      in.LocalReference,
		ReferenceName:       in.ReferenceName,
		GlobalAccess:        internal.NewAccessSpecRef(in.GlobalAccess),
		MediaType:           in.MediaType,
	}, nil
}

func (_ converterV1) ConvertTo(object interface{}) (cpi.AccessSpec, error) {
	in, ok := object.(*AccessSpecV1)
	if !ok {
		return nil, fmt.Errorf("failed to assert type %T to AccessSpecV1", object)
	}
	return &AccessSpec{
		ObjectVersionedType: runtime.NewVersionedObjectType(in.Type),
		LocalReference:      in.LocalReference,
		ReferenceName:       in.ReferenceName,
		GlobalAccess:        in.GlobalAccess,
		MediaType:           in.MediaType,
	}, nil
}
