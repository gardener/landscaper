// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package localblob

import (
	"encoding/json"
	"fmt"

	. "github.com/open-component-model/ocm/pkg/exception"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/mime"
	"github.com/open-component-model/ocm/pkg/runtime"
)

// Type is the access type of a blob local to a component.
const (
	Type   = "localBlob"
	TypeV1 = Type + runtime.VersionSeparator + "v1"
)

// this package shows how to implement access types with multiple serialization versions.
// So far, only one is implemented, but it shows how to add other ones.
//
// Specifications using multiple format versions allways provide a single common
// *internal* Go representation, intended to be used by library users. Only this
// internal version should be used outside this package. Additionally, there
// are Go types representing the various format versions, which will be used
// for the de-/serialization process (here AccessSpecV1).
//
// The supported versions are gathered in a dedicated scheme object (variable versions),
// which is then used to register all available versions at the default scheme (see
// init method).
// The *internal* specification Go type (here AccessSpec) must be based on
// runtime.InternalVersionedObjectType.
// It is initialized with the effective type/version name and the versions scheme
// and represents the Go representation used by API users, the format versions
// are never used outside this package.
//
// Additionally, this *internal* type must implement the MarshalJSON method, which
// can be implemented by delegating to the runtime.MarshalVersionedTypedObject
// method, which evaluated the versions scheme to finds the applicable conversion
// provided by the runtime.InternalVersionedObjectType.
//
// For every format version runtime.FormatVersion is required, which can be created
// with cpi.NewAccessSpecVersion, which takes the prototype and a converter,
// which converts between the internal go representation and the external formats,
// given by a dedicated go Type with serialization annotations.

var versions = cpi.NewAccessTypeVersionScheme(Type)

func init() {
	Must(versions.Register(cpi.NewAccessSpecTypeByConverter[*AccessSpec, *AccessSpecV1](Type, &converterV1{}, cpi.WithDescription(usage))))
	Must(versions.Register(cpi.NewAccessSpecTypeByConverter[*AccessSpec, *AccessSpecV1](TypeV1, &converterV1{}, cpi.WithFormatSpec(formatV1), cpi.WithConfigHandler(ConfigHandler()))))
	cpi.RegisterAccessTypeVersions(versions)
}

func Is(spec cpi.AccessSpec) bool {
	return spec != nil && spec.GetKind() == Type
}

// New creates a new localFilesystemBlob accessor.
func New(local, hint string, mediaType string, global cpi.AccessSpec) *AccessSpec {
	return &AccessSpec{
		InternalVersionedTypedObject: runtime.NewInternalVersionedTypedObject[cpi.AccessSpec](versions, Type),
		LocalReference:               local,
		ReferenceName:                hint,
		MediaType:                    mediaType,
		GlobalAccess:                 cpi.NewAccessSpecRef(global),
	}
}

func Decode(data []byte) (*AccessSpec, error) {
	spec, err := versions.Decode(data, runtime.DefaultYAMLEncoding)
	if err != nil {
		return nil, err
	}
	return spec.(*AccessSpec), nil
}

// AccessSpec describes the access for a local blob.
type AccessSpec struct {
	runtime.InternalVersionedTypedObject[cpi.AccessSpec]
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
	return runtime.MarshalVersionedTypedObject(&a)
	// return cpi.MarshalConvertedAccessSpec(cpi.DefaultContext(), &a)
}

func (a *AccessSpec) Describe(ctx cpi.Context) string {
	return fmt.Sprintf("Local blob %s[%s]", a.LocalReference, a.ReferenceName)
}

func (a *AccessSpec) IsLocal(cpi.Context) bool {
	return true
}

func (a *AccessSpec) GlobalAccessSpec(ctx cpi.Context) cpi.AccessSpec {
	if g, err := ctx.AccessSpecForSpec(a.GlobalAccess); err == nil {
		return g
	}
	return a.GlobalAccess
}

func (a *AccessSpec) GetMimeType() string {
	if a.MediaType == "" {
		return mime.MIME_OCTET
	}
	return a.MediaType
}

func (a *AccessSpec) GetReferenceHint(cv cpi.ComponentVersionAccess) string {
	return a.ReferenceName
}

func (a *AccessSpec) AccessMethod(cv cpi.ComponentVersionAccess) (cpi.AccessMethod, error) {
	return cv.AccessMethod(a)
}

func (a *AccessSpec) GetInexpensiveContentVersionIdentity(cv cpi.ComponentVersionAccess) string {
	return cv.GetInexpensiveContentVersionIdentity(a)
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

type converterV1 struct{}

func (_ converterV1) ConvertFrom(in *AccessSpec) (*AccessSpecV1, error) {
	return &AccessSpecV1{
		ObjectVersionedType: runtime.NewVersionedTypedObject(in.Type),
		LocalReference:      in.LocalReference,
		ReferenceName:       in.ReferenceName,
		GlobalAccess:        cpi.NewAccessSpecRef(in.GlobalAccess),
		MediaType:           in.MediaType,
	}, nil
}

func (_ converterV1) ConvertTo(in *AccessSpecV1) (*AccessSpec, error) {
	return &AccessSpec{
		InternalVersionedTypedObject: runtime.NewInternalVersionedTypedObject[cpi.AccessSpec](versions, in.Type),
		LocalReference:               in.LocalReference,
		ReferenceName:                in.ReferenceName,
		GlobalAccess:                 in.GlobalAccess,
		MediaType:                    in.MediaType,
	}, nil
}
