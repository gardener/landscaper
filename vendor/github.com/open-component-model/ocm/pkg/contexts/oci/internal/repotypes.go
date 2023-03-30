// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/modern-go/reflect2"

	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

type RepositoryType interface {
	runtime.TypedObjectDecoder
	runtime.VersionedTypedObject
}

type IntermediateRepositorySpecAspect interface {
	IsIntermediate() bool
}

type RepositorySpec interface {
	runtime.VersionedTypedObject

	Name() string
	UniformRepositorySpec() *UniformRepositorySpec
	Repository(Context, credentials.Credentials) (Repository, error)
}

type RepositoryTypeScheme interface {
	runtime.Scheme
	AddKnownTypes(s RepositoryTypeScheme)

	GetRepositoryType(name string) RepositoryType
	Register(name string, atype RepositoryType)

	DecodeRepositorySpec(data []byte, unmarshaler runtime.Unmarshaler) (RepositorySpec, error)
	CreateRepositorySpec(obj runtime.TypedObject) (RepositorySpec, error)
}

type repositoryTypeScheme struct {
	runtime.SchemeBase
}

func NewRepositoryTypeScheme(defaultRepoDecoder runtime.TypedObjectDecoder, base ...RepositoryTypeScheme) RepositoryTypeScheme {
	var rt RepositorySpec
	scheme := runtime.MustNewDefaultScheme(&rt, &UnknownRepositorySpec{}, true, defaultRepoDecoder, utils.Optional(base...))
	return &repositoryTypeScheme{scheme}
}

func (t *repositoryTypeScheme) AddKnownTypes(s RepositoryTypeScheme) {
	t.SchemeBase.AddKnownTypes(s)
}

func (t *repositoryTypeScheme) GetRepositoryType(name string) RepositoryType {
	d := t.GetDecoder(name)
	if d == nil {
		return nil
	}
	return d.(RepositoryType)
}

func (t *repositoryTypeScheme) RegisterByDecoder(name string, decoder runtime.TypedObjectDecoder) error {
	if _, ok := decoder.(RepositoryType); !ok {
		return errors.ErrInvalid("type", reflect.TypeOf(decoder).String())
	}
	return t.SchemeBase.RegisterByDecoder(name, decoder)
}

func (t *repositoryTypeScheme) Register(name string, rtype RepositoryType) {
	t.SchemeBase.RegisterByDecoder(name, rtype)
}

func (t *repositoryTypeScheme) DecodeRepositorySpec(data []byte, unmarshaler runtime.Unmarshaler) (RepositorySpec, error) {
	obj, err := t.Decode(data, unmarshaler)
	if err != nil {
		return nil, err
	}
	if spec, ok := obj.(RepositorySpec); ok {
		return spec, nil
	}
	return nil, fmt.Errorf("invalid access spec type: yield %T instead of RepositorySpec", obj)
}

func (t *repositoryTypeScheme) CreateRepositorySpec(obj runtime.TypedObject) (RepositorySpec, error) {
	if s, ok := obj.(RepositorySpec); ok {
		r, err := t.SchemeBase.Convert(s)
		if err != nil {
			return nil, err
		}
		return r.(RepositorySpec), nil
	}
	if u, ok := obj.(*runtime.UnstructuredTypedObject); ok {
		raw, err := u.GetRaw()
		if err != nil {
			return nil, err
		}
		return t.DecodeRepositorySpec(raw, runtime.DefaultJSONEncoding)
	}
	return nil, fmt.Errorf("invalid object type %T for repository specs", obj)
}

// DefaultRepositoryTypeScheme contains all globally known access serializer.
var DefaultRepositoryTypeScheme = NewRepositoryTypeScheme(nil)

func RegisterRepositoryType(name string, atype RepositoryType) {
	DefaultRepositoryTypeScheme.Register(name, atype)
}

func CreateRepositorySpec(t runtime.TypedObject) (RepositorySpec, error) {
	return DefaultRepositoryTypeScheme.CreateRepositorySpec(t)
}

type UnknownRepositorySpec struct {
	runtime.UnstructuredVersionedTypedObject `json:",inline"`
}

var _ RepositorySpec = &UnknownRepositorySpec{}

func (r *UnknownRepositorySpec) Name() string {
	return "unknown-" + r.GetKind()
}

func (r *UnknownRepositorySpec) UniformRepositorySpec() *UniformRepositorySpec {
	return UniformRepositorySpecForUnstructured(&r.UnstructuredVersionedTypedObject)
}

func (r *UnknownRepositorySpec) Repository(Context, credentials.Credentials) (Repository, error) {
	return nil, errors.ErrUnknown("repository type", r.GetType())
}

////////////////////////////////////////////////////////////////////////////////

type GenericRepositorySpec struct {
	runtime.UnstructuredVersionedTypedObject `json:",inline"`
}

func (s *GenericRepositorySpec) Name() string {
	return "generic-" + s.GetKind()
}

func (s *GenericRepositorySpec) UniformRepositorySpec() *UniformRepositorySpec {
	return UniformRepositorySpecForUnstructured(&s.UnstructuredVersionedTypedObject)
}

func (s *GenericRepositorySpec) Evaluate(ctx Context) (RepositorySpec, error) {
	raw, err := s.GetRaw()
	if err != nil {
		return nil, err
	}
	return ctx.RepositoryTypes().DecodeRepositorySpec(raw, runtime.DefaultJSONEncoding)
}

func (s *GenericRepositorySpec) Repository(ctx Context, creds credentials.Credentials) (Repository, error) {
	spec, err := s.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	return spec.Repository(ctx, creds)
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
	s, err := newGenericRepositorySpec(data, unmarshaler)
	if err != nil {
		return nil, err // GO is great
	}
	return s, nil
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

////////////////////////////////////////////////////////////////////////////////
