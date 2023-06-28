// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"encoding/json"
	"fmt"

	"github.com/modern-go/reflect2"

	"github.com/open-component-model/ocm/pkg/cobrautils/flagsets"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/generics"
	"github.com/open-component-model/ocm/pkg/logging"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

type AccessType interface {
	runtime.VersionedTypedObjectType[AccessSpec]

	ConfigOptionTypeSetHandler() flagsets.ConfigOptionTypeSetHandler

	Description() string
	Format() string
}

type AccessMethodSupport interface {
	GetContext() Context
	LocalSupportForAccessSpec(spec AccessSpec) bool
}

// AccessSpec is the interface access method specifications
// must fulfill. The main task is to map the specification
// to a concrete implementation of the access method for a dedicated
// component version.
type AccessSpec interface {
	compdesc.AccessSpec
	Describe(Context) string
	IsLocal(Context) bool
	GlobalAccessSpec(Context) AccessSpec
	// AccessMethod provides an access method implementation for
	// an access spec. This might be a repository local implementation
	// or a global one. It might be implemented directly by the AccessSpec
	// for global AccessMethods or forwarded to the ComponentVersion for
	// local access methods. It may only be forwarded for AccessSpecs stating
	// to be local (IsLocal()==true).
	// This forwarding is necessary because the concrete implementation of
	// the currently used OCM Repository is not known to the AccessSpec.
	AccessMethod(access ComponentVersionAccess) (AccessMethod, error)
	// GetInexpensiveContentVersionIdentity implements a method that attempts to provide an inexpensive identity.
	// Therefore, an identity that can be provided without requiring the entire object (e.g. calculating the digest from
	// the bytes), which would defeat the purpose of caching.
	// It follows the same contract as AccessMethod.
	GetInexpensiveContentVersionIdentity(access ComponentVersionAccess) string
}

type (
	AccessSpecDecoder  = runtime.TypedObjectDecoder[AccessSpec]
	AccessTypeProvider = runtime.KnownTypesProvider[AccessSpec, AccessType]
)

// HintProvider is used to provide a reference hint for local access method specs.
// It may optionally be provided by an access spec.
// When adding blobs to a repository the hint is used by blobhandlers for
// expanding a blob to a repository specific representation to determine a
// useful name.
type HintProvider interface {
	GetReferenceHint(cv ComponentVersionAccess) string
}

// AccessMethod described the access to a dedicated resource
// It can allocate external resources, which should be released
// with the Close() call.
// Resources SHOULD only be allocated, if the content is accessed
// via the DataAccess interface to avoid unnecessary effort
// if the method object is just used to access meta data.
type AccessMethod interface {
	DataAccess

	GetKind() string
	AccessSpec() AccessSpec
	MimeType
	Close() error
}

type AccessTypeScheme interface {
	runtime.TypeScheme[AccessSpec, AccessType]

	CreateConfigTypeSetConfigProvider() flagsets.ConfigTypeOptionSetConfigProvider
}

type _AccessScheme = runtime.TypeScheme[AccessSpec, AccessType]

type accessTypeScheme struct {
	_AccessScheme
}

func NewAccessTypeScheme(base ...AccessTypeScheme) AccessTypeScheme {
	scheme := runtime.MustNewDefaultTypeScheme[AccessSpec, AccessType](&UnknownAccessSpec{}, true, nil, utils.Optional(base...))
	return &accessTypeScheme{scheme}
}

func NewStrictAccessTypeScheme(base ...AccessTypeScheme) runtime.VersionedTypeRegistry[AccessSpec, AccessType] {
	scheme := runtime.MustNewDefaultTypeScheme[AccessSpec, AccessType](nil, false, nil, utils.Optional(base...))
	return &accessTypeScheme{scheme}
}

func (t *accessTypeScheme) CreateConfigTypeSetConfigProvider() flagsets.ConfigTypeOptionSetConfigProvider {
	prov := flagsets.NewTypedConfigProvider("access", "blob access specification")
	prov.AddGroups("Access Specification Options")
	for _, p := range t.KnownTypes() {
		err := prov.AddTypeSet(p.ConfigOptionTypeSetHandler())
		if err != nil {
			logging.Logger().LogError(err, "cannot compose access type CLI options")
		}
	}
	if t.BaseScheme() != nil {
		base := t.BaseScheme().(AccessTypeScheme)
		for _, s := range base.CreateConfigTypeSetConfigProvider().OptionTypeSets() {
			if prov.GetTypeSet(s.GetName()) == nil {
				err := prov.AddTypeSet(s)
				if err != nil {
					logging.Logger().LogError(err, "cannot compose access type CLI options")
				}
			}
		}
	}

	return prov
}

func (t *accessTypeScheme) KnownTypes() runtime.KnownTypes[AccessSpec, AccessType] {
	return t._AccessScheme.KnownTypes() // Goland
}

// DefaultAccessTypeScheme contains all globally known access serializer.
var DefaultAccessTypeScheme = NewAccessTypeScheme()

func RegisterAccessType(atype AccessType) {
	DefaultAccessTypeScheme.Register(atype)
}

func CreateAccessSpec(t runtime.TypedObject) (AccessSpec, error) {
	return DefaultAccessTypeScheme.Convert(t)
}

////////////////////////////////////////////////////////////////////////////////

type UnknownAccessSpec struct {
	runtime.UnstructuredVersionedTypedObject `json:",inline"`
}

var (
	_ runtime.TypedObject = &UnknownAccessSpec{}
	_ runtime.Unknown     = &UnknownAccessSpec{}
)

func (_ *UnknownAccessSpec) IsUnknown() bool {
	return true
}

func (s *UnknownAccessSpec) AccessMethod(ComponentVersionAccess) (AccessMethod, error) {
	return nil, errors.ErrUnknown(errors.KIND_ACCESSMETHOD, s.GetType())
}

func (s *UnknownAccessSpec) GetInexpensiveContentVersionIdentity(ComponentVersionAccess) string {
	return ""
}

func (s *UnknownAccessSpec) Describe(ctx Context) string {
	return fmt.Sprintf("unknown access method type %q", s.GetType())
}

func (_ *UnknownAccessSpec) IsLocal(Context) bool {
	return false
}

func (_ *UnknownAccessSpec) GlobalAccessSpec(Context) AccessSpec {
	return nil
}

var _ AccessSpec = &UnknownAccessSpec{}

////////////////////////////////////////////////////////////////////////////////

type EvaluatableAccessSpec interface {
	AccessSpec
	Evaluate(ctx Context) (AccessSpec, error)
}

type GenericAccessSpec struct {
	runtime.UnstructuredVersionedTypedObject `json:",inline"`
}

var _ AccessSpec = &GenericAccessSpec{}

func ToGenericAccessSpec(spec AccessSpec) (*GenericAccessSpec, error) {
	if reflect2.IsNil(spec) {
		return nil, nil
	}
	if g, ok := spec.(*GenericAccessSpec); ok {
		return g, nil
	}
	data, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	return newGenericAccessSpec(data, runtime.DefaultJSONEncoding)
}

func NewGenericAccessSpec(data []byte, unmarshaler ...runtime.Unmarshaler) (AccessSpec, error) {
	return generics.AsE[AccessSpec](newGenericAccessSpec(data, utils.Optional(unmarshaler...)))
}

func newGenericAccessSpec(data []byte, unmarshaler runtime.Unmarshaler) (*GenericAccessSpec, error) {
	unstr := &runtime.UnstructuredVersionedTypedObject{}
	if unmarshaler == nil {
		unmarshaler = runtime.DefaultYAMLEncoding
	}
	err := unmarshaler.Unmarshal(data, unstr)
	if err != nil {
		return nil, err
	}
	return &GenericAccessSpec{*unstr}, nil
}

func (s *GenericAccessSpec) Describe(ctx Context) string {
	eff, err := s.Evaluate(ctx)
	if err != nil {
		return fmt.Sprintf("invalid access specification: %s", err.Error())
	}
	return eff.Describe(ctx)
}

func (s *GenericAccessSpec) Evaluate(ctx Context) (AccessSpec, error) {
	raw, err := s.GetRaw()
	if err != nil {
		return nil, err
	}
	return ctx.AccessMethods().Decode(raw, runtime.DefaultJSONEncoding)
}

func (s *GenericAccessSpec) AccessMethod(acc ComponentVersionAccess) (AccessMethod, error) {
	spec, err := s.Evaluate(acc.GetContext())
	if err != nil {
		return nil, err
	}
	if _, ok := spec.(*GenericAccessSpec); ok {
		return nil, errors.ErrUnknown(errors.KIND_ACCESSMETHOD, s.GetType())
	}
	return spec.AccessMethod(acc)
}

func (s *GenericAccessSpec) GetInexpensiveContentVersionIdentity(acc ComponentVersionAccess) string {
	spec, err := s.Evaluate(acc.GetContext())
	if err != nil {
		return ""
	}
	if _, ok := spec.(*GenericAccessSpec); ok {
		return ""
	}
	return spec.GetInexpensiveContentVersionIdentity(acc)
}

func (s *GenericAccessSpec) IsLocal(ctx Context) bool {
	spec, err := s.Evaluate(ctx)
	if err != nil {
		return false
	}
	if _, ok := spec.(*GenericAccessSpec); ok {
		return false
	}
	return spec.IsLocal(ctx)
}

func (s *GenericAccessSpec) GlobalAccessSpec(ctx Context) AccessSpec {
	spec, err := s.Evaluate(ctx)
	if err != nil {
		return nil
	}
	if _, ok := spec.(*GenericAccessSpec); ok {
		return nil
	}
	return spec.GlobalAccessSpec(ctx)
}

////////////////////////////////////////////////////////////////////////////////
