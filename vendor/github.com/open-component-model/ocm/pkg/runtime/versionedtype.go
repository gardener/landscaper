// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"strings"

	"golang.org/x/exp/slices"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
)

const VersionSeparator = "/"

// VersionedTypeInfo in the accessor for versioned type information.
type VersionedTypeInfo interface {
	TypeInfo
	GetKind() string
	GetVersion() string
}

// VersionedTypedObject in an instance of a VersionedType.
type VersionedTypedObject interface {
	TypedObject
	VersionedTypeInfo
}

////////////////////////////////////////////////////////////////////////////////
// Versioned Typed Objects

// ObjectVersionedTypedObject is a minimal implementation of a VersionedTypedObject.
type ObjectVersionedTypedObject = VersionedObjectType

// ObjectVersionedType is a minimal implementation of a VersionedTypedObject.
// For compatibility, we keep the old not aligned type name.
type ObjectVersionedType = ObjectVersionedTypedObject

// NewVersionedTypedObject creates an ObjectVersionedType value.
func NewVersionedTypedObject(args ...string) ObjectVersionedTypedObject {
	return ObjectVersionedTypedObject{Type: TypeName(args...)}
}

////////////////////////////////////////////////////////////////////////////////
// Object Types for Versioned Typed Objects

// InternalVersionedTypedObject is the base type used
// by *internal* representations of versioned specification
// formats. It is used to convert from/to dedicated
// format versions.
type InternalVersionedTypedObject[T VersionedTypedObject] struct {
	ObjectVersionedType
	encoder TypedObjectEncoder[T]
}

var _ encoder = (*InternalVersionedTypedObject[VersionedTypedObject])(nil)

type encoder interface {
	encode(obj VersionedTypedObject) ([]byte, error)
}

func NewInternalVersionedTypedObject[T VersionedTypedObject](encoder TypedObjectEncoder[T], types ...string) InternalVersionedTypedObject[T] {
	return InternalVersionedTypedObject[T]{
		ObjectVersionedType: NewVersionedObjectType(types...),
		encoder:             encoder,
	}
}

func (o *InternalVersionedTypedObject[T]) encode(obj VersionedTypedObject) ([]byte, error) {
	// cannot use type parameter here, because casts of paramerized objects are not supported in GO
	return o.encoder.Encode(obj.(T), DefaultJSONEncoding)
}

func GetEncoder[T VersionedTypedObject](obj T) encoder {
	var i interface{} = obj
	if e, ok := i.(encoder); ok {
		return e
	}
	return nil
}

func MarshalVersionedTypedObject[T VersionedTypedObject](obj T, toe ...TypedObjectEncoder[T]) ([]byte, error) {
	if e := GetEncoder(obj); e != nil {
		return e.encode(obj)
	}
	if e := utils.Optional(toe...); e != nil {
		return e.Encode(obj, DefaultJSONEncoding)
	}
	return nil, errors.ErrUnknown("object type", obj.GetType())
}

////////////////////////////////////////////////////////////////////////////////

// VersionedTypedObjectType is the interface of a type object for a versioned type.
type VersionedTypedObjectType[T VersionedTypedObject] interface {
	VersionedTypeInfo
	TypedObjectDecoder[T]
	TypedObjectEncoder[T]
}

type versionedTypedObjectType[T VersionedTypedObject] struct {
	_VersionedObjectType
	_FormatVersion[T]
}

var _ FormatVersion[VersionedTypedObject] = (*versionedTypedObjectType[VersionedTypedObject])(nil)

func NewVersionedTypedObjectType[T VersionedTypedObject, I VersionedTypedObject](name string) VersionedTypedObjectType[T] {
	return &versionedTypedObjectType[T]{
		_VersionedObjectType: NewVersionedObjectType(name),
		_FormatVersion:       NewSimpleVersion[T, I](),
	}
}

func NewVersionedTypedObjectTypeByProtoConverter[T VersionedTypedObject, I VersionedTypedObject](name string, proto TypedObject, converter Converter[I, TypedObject]) VersionedTypedObjectType[T] {
	return &versionedTypedObjectType[T]{
		_VersionedObjectType: NewVersionedObjectType(name),
		_FormatVersion:       NewConvertedVersionByProto[T, I](proto, converter),
	}
}

func NewVersionedTypedObjectTypeByConverter[T VersionedTypedObject, I VersionedTypedObject, V TypedObject](name string, converter Converter[I, V]) VersionedTypedObjectType[T] {
	return &versionedTypedObjectType[T]{
		_VersionedObjectType: NewVersionedObjectType(name),
		_FormatVersion:       NewConvertedVersion[T, I, V](converter),
	}
}

func NewVersionedTypedObjectTypeByFormatVersion[T VersionedTypedObject](name string, fmt FormatVersion[T]) VersionedTypedObjectType[T] {
	return &versionedTypedObjectType[T]{
		_VersionedObjectType: NewVersionedObjectType(name),
		_FormatVersion:       fmt,
	}
}

// NewVersionedTypedObjectTypeByVersion creates a new type object for versioned typed objects,
// where T is the common *interface* of all types of the same type realm and I is the
// *internal implementation* commonly used for the various version variants of a dedicated kind of type.
// Therefore, I must be subtype of T, which cannot be expressed in Go.
func NewVersionedTypedObjectTypeByVersion[T VersionedTypedObject, I VersionedTypedObject](name string, version FormatVersion[I]) VersionedTypedObjectType[T] {
	return &versionedTypedObjectType[T]{
		_VersionedObjectType: NewVersionedObjectType(name),
		_FormatVersion:       &caster[T, I]{version},
	}
}

////////////////////////////////////////////////////////////////////////////////

type versionedTypeRegistryBase[T VersionedTypedObject, R VersionedTypedObjectType[T]] interface {
	KnownTypesProvider[T, R]
	TypedObjectEncoder[T]
	TypedObjectDecoder[T]
}

type VersionedTypeRegistry[T VersionedTypedObject, R VersionedTypedObjectType[T]] interface {
	Register(t R)
	versionedTypeRegistryBase[T, R]
}

// TypeVersionScheme is used to register different versions for the same internal
// representation of a versioned typed object.
type TypeVersionScheme[T VersionedTypedObject, R VersionedTypedObjectType[T]] interface {
	WithKindAliases(kind ...string) TypeVersionScheme[T, R]
	Register(t R) error
	versionedTypeRegistryBase[T, R]
}

type typeVersionScheme[T VersionedTypedObject, R VersionedTypedObjectType[T]] struct {
	kind                            string
	aliases                         []string
	base                            VersionedTypeRegistry[T, R]
	versionedTypeRegistryBase[T, R] // required for Goland, cannot handle overwrite of Register method with different signature
}

func NewTypeVersionScheme[T VersionedTypedObject, R VersionedTypedObjectType[T]](kind string, reg VersionedTypeRegistry[T, R]) TypeVersionScheme[T, R] {
	return &typeVersionScheme[T, R]{kind, nil, reg, reg}
}

func (s *typeVersionScheme[T, R]) WithKindAliases(kind ...string) TypeVersionScheme[T, R] {
	s.aliases = append(s.aliases, kind...)
	return s
}

func (s *typeVersionScheme[T, R]) Register(t R) error {
	if t.GetKind() != s.kind && !slices.Contains(s.aliases, t.GetKind()) {
		return errors.ErrInvalid("repository spec type", t.GetType(), "kind", s.kind)
	}

	s.base.Register(t)
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func TypeName(args ...string) string {
	if len(args) == 1 {
		return args[0]
	}
	if len(args) == 2 {
		if args[1] == "" {
			return args[0]
		}
		return args[0] + VersionSeparator + args[1]
	}
	panic("invalid call to TypeName, one or two arguments required")
}

func KindVersion(t string) (string, string) {
	i := strings.LastIndex(t, VersionSeparator)
	if i > 0 {
		return t[:i], t[i+1:]
	}
	return t, ""
}

func GetKind(v TypedObject) string {
	t := v.GetType()
	i := strings.LastIndex(t, VersionSeparator)
	if i < 0 {
		return t
	}
	return t[:i]
}

func GetVersion(v TypedObject) string {
	t := v.GetType()
	i := strings.LastIndex(t, VersionSeparator)
	if i < 0 {
		return "v1"
	}
	return t[i+1:]
}
