// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"encoding/json"

	"github.com/modern-go/reflect2"

	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/generics"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

type RepositoryType interface {
	runtime.VersionedTypedObjectType[RepositorySpec]

	// LocalSupportForAccessSpec checks whether a repository
	// provides a local version for the access spec.
	// LocalSupportForAccessSpec(ctx Context, a compdesc.AccessSpec) bool
}

type IntermediateRepositorySpecAspect = oci.IntermediateRepositorySpecAspect

type RepositorySpec interface {
	runtime.VersionedTypedObject

	AsUniformSpec(Context) *UniformRepositorySpec
	Repository(Context, credentials.Credentials) (Repository, error)
}

type (
	RepositorySpecDecoder  = runtime.TypedObjectDecoder[RepositorySpec]
	RepositoryTypeProvider = runtime.KnownTypesProvider[RepositorySpec, RepositoryType]
)

type RepositoryTypeScheme interface {
	runtime.TypeScheme[RepositorySpec, RepositoryType]
}

type _Scheme = runtime.TypeScheme[RepositorySpec, RepositoryType]

type repositoryTypeScheme struct {
	_Scheme
}

func NewRepositoryTypeScheme(defaultDecoder RepositorySpecDecoder, base ...RepositoryTypeScheme) RepositoryTypeScheme {
	scheme := runtime.MustNewDefaultTypeScheme[RepositorySpec, RepositoryType](&UnknownRepositorySpec{}, true, defaultDecoder, utils.Optional(base...))
	return &repositoryTypeScheme{scheme}
}

func NewStrictRepositoryTypeScheme(base ...RepositoryTypeScheme) RepositoryTypeScheme {
	scheme := runtime.MustNewDefaultTypeScheme[RepositorySpec, RepositoryType](nil, false, nil, utils.Optional(base...))
	return &repositoryTypeScheme{scheme}
}

func (t *repositoryTypeScheme) KnownTypes() runtime.KnownTypes[RepositorySpec, RepositoryType] {
	return t._Scheme.KnownTypes() // Goland
}

// DefaultRepositoryTypeScheme contains all globally known access serializer.
var DefaultRepositoryTypeScheme = NewRepositoryTypeScheme(nil)

func RegisterRepositoryType(atype RepositoryType) {
	DefaultRepositoryTypeScheme.Register(atype)
}

func CreateRepositorySpec(t runtime.TypedObject) (RepositorySpec, error) {
	return DefaultRepositoryTypeScheme.Convert(t)
}

////////////////////////////////////////////////////////////////////////////////

type UnknownRepositorySpec struct {
	runtime.UnstructuredVersionedTypedObject `json:",inline"`
}

var (
	_ RepositorySpec  = &UnknownRepositorySpec{}
	_ runtime.Unknown = &UnknownRepositorySpec{}
)

func (_ *UnknownRepositorySpec) IsUnknown() bool {
	return true
}

func (r *UnknownRepositorySpec) AsUniformSpec(Context) *UniformRepositorySpec {
	return UniformRepositorySpecForUnstructured(&r.UnstructuredVersionedTypedObject)
}

func (r *UnknownRepositorySpec) Repository(Context, credentials.Credentials) (Repository, error) {
	return nil, errors.ErrUnknown("repository type", r.GetType())
}

////////////////////////////////////////////////////////////////////////////////

type GenericRepositorySpec struct {
	runtime.UnstructuredVersionedTypedObject `json:",inline"`
}

var _ RepositorySpec = &GenericRepositorySpec{}

func ToGenericRepositorySpec(spec RepositorySpec) (*GenericRepositorySpec, error) {
	if reflect2.IsNil(spec) {
		return nil, nil
	}
	if g, ok := spec.(*GenericRepositorySpec); ok {
		return g, nil
	}
	data, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	return newGenericRepositorySpec(data, runtime.DefaultJSONEncoding)
}

func NewGenericRepositorySpec(data []byte, unmarshaler runtime.Unmarshaler) (RepositorySpec, error) {
	return generics.AsE[RepositorySpec](newGenericRepositorySpec(data, unmarshaler))
}

func newGenericRepositorySpec(data []byte, unmarshaler runtime.Unmarshaler) (*GenericRepositorySpec, error) {
	unstr := &runtime.UnstructuredVersionedTypedObject{}
	if unmarshaler == nil {
		unmarshaler = runtime.DefaultYAMLEncoding
	}
	err := unmarshaler.Unmarshal(data, unstr)
	if err != nil {
		return nil, err
	}
	return &GenericRepositorySpec{*unstr}, nil
}

func (s *GenericRepositorySpec) AsUniformSpec(ctx Context) *UniformRepositorySpec {
	eff, err := s.Evaluate(ctx)
	if err != nil {
		return &UniformRepositorySpec{Type: s.GetKind()}
	}
	return eff.AsUniformSpec(ctx)
}

func (s *GenericRepositorySpec) Evaluate(ctx Context) (RepositorySpec, error) {
	raw, err := s.GetRaw()
	if err != nil {
		return nil, err
	}
	return ctx.RepositoryTypes().Decode(raw, runtime.DefaultJSONEncoding)
}

func (s *GenericRepositorySpec) Repository(ctx Context, creds credentials.Credentials) (Repository, error) {
	spec, err := s.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	return spec.Repository(ctx, creds)
}

////////////////////////////////////////////////////////////////////////////////
