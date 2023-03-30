// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
	"reflect"

	"github.com/open-component-model/ocm/pkg/cobrautils/flagsets"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/logging"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

type AccessType interface {
	runtime.TypedObjectDecoder
	runtime.VersionedTypedObject

	ConfigOptionTypeSetHandler() flagsets.ConfigOptionTypeSetHandler

	Description() string
	Format(cli bool) string
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
	AccessMethod(access ComponentVersionAccess) (AccessMethod, error)
}

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
	runtime.Scheme
	AddKnownTypes(s AccessTypeScheme)

	GetAccessType(name string) AccessType
	Register(name string, atype AccessType)

	DecodeAccessSpec(data []byte, unmarshaler runtime.Unmarshaler) (AccessSpec, error)
	CreateAccessSpec(obj runtime.TypedObject) (AccessSpec, error)

	CreateConfigTypeSetConfigProvider() flagsets.ConfigTypeOptionSetConfigProvider
}

type accessTypeScheme struct {
	runtime.SchemeBase
	base        AccessTypeScheme
	optionTypes map[string]AccessType
}

func NewAccessTypeScheme(base ...AccessTypeScheme) AccessTypeScheme {
	var at AccessSpec
	b := utils.Optional(base...)
	scheme := runtime.MustNewDefaultScheme(&at, &UnknownAccessSpec{}, true, nil, b)
	return &accessTypeScheme{scheme, b, map[string]AccessType{}}
}

func (t *accessTypeScheme) AddKnownTypes(s AccessTypeScheme) {
	t.SchemeBase.AddKnownTypes(s)
}

func (t *accessTypeScheme) CreateConfigTypeSetConfigProvider() flagsets.ConfigTypeOptionSetConfigProvider {
	prov := flagsets.NewTypedConfigProvider("access", "blob access specification")
	prov.AddGroups("Access Specification Options")
	for _, p := range t.optionTypes {
		err := prov.AddTypeSet(p.ConfigOptionTypeSetHandler())
		if err != nil {
			logging.Logger().LogError(err, "cannot compose access type CLI options")
		}
	}
	if t.base != nil {
		for _, s := range t.base.CreateConfigTypeSetConfigProvider().OptionTypeSets() {
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

func (t *accessTypeScheme) GetAccessType(name string) AccessType {
	decoder := t.GetDecoder(name)
	if decoder == nil {
		return nil
	}
	return decoder.(AccessType)
}

func (t *accessTypeScheme) Register(name string, atype AccessType) {
	t.SchemeBase.RegisterByDecoder(name, atype)
	t.optionTypes[atype.GetType()] = atype
}

func (t *accessTypeScheme) RegisterByDecoder(name string, decoder runtime.TypedObjectDecoder) error {
	if atype, ok := decoder.(AccessType); !ok {
		return errors.ErrInvalid("type", reflect.TypeOf(decoder).String())
	} else {
		t.Register(name, atype)
	}
	return nil
}

func (t *accessTypeScheme) DecodeAccessSpec(data []byte, unmarshaler runtime.Unmarshaler) (AccessSpec, error) {
	obj, err := t.Decode(data, unmarshaler)
	if err != nil {
		return nil, err
	}
	if spec, ok := obj.(AccessSpec); ok {
		return spec, nil
	}
	return nil, fmt.Errorf("invalid access spec type: yield %T instead of AccessSpec", obj)
}

func (t *accessTypeScheme) CreateAccessSpec(obj runtime.TypedObject) (AccessSpec, error) {
	if s, ok := obj.(AccessSpec); ok {
		return s, nil
	}
	if u, ok := obj.(*runtime.UnstructuredTypedObject); ok {
		raw, err := u.GetRaw()
		if err != nil {
			return nil, err
		}
		return t.DecodeAccessSpec(raw, runtime.DefaultJSONEncoding)
	}
	return nil, errors.ErrInvalid("object type", fmt.Sprintf("%T", obj), "access specs")
}

// DefaultAccessTypeScheme contains all globally known access serializer.
var DefaultAccessTypeScheme = NewAccessTypeScheme()

func GetAccessType(name string) AccessType {
	return DefaultAccessTypeScheme.GetAccessType(name)
}

func CreateAccessSpec(t runtime.TypedObject) (AccessSpec, error) {
	return DefaultAccessTypeScheme.CreateAccessSpec(t)
}

////////////////////////////////////////////////////////////////////////////////

type UnknownAccessSpec struct {
	runtime.UnstructuredVersionedTypedObject `json:",inline"`
}

var _ runtime.TypedObject = &UnknownAccessSpec{}

func (s *UnknownAccessSpec) AccessMethod(ComponentVersionAccess) (AccessMethod, error) {
	return nil, errors.ErrUnknown(errors.KIND_ACCESSMETHOD, s.GetType())
}

func (s *UnknownAccessSpec) Describe(ctx Context) string {
	return fmt.Sprintf("unknown access method type %q", s.GetType())
}

func (_ *UnknownAccessSpec) IsLocal(Context) bool {
	return false
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

func NewGenericAccessSpec(spec string) (*GenericAccessSpec, error) {
	var g GenericAccessSpec
	err := runtime.DefaultYAMLEncoding.Unmarshal([]byte(spec), &g)
	if err != nil {
		return nil, err
	}
	return &g, nil
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
	return ctx.AccessMethods().DecodeAccessSpec(raw, runtime.DefaultJSONEncoding)
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

var _ AccessSpec = &GenericAccessSpec{}
