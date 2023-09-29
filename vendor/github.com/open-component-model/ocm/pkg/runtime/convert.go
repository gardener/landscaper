// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"fmt"
	"reflect"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/generics"
)

type Converter[T VersionedTypedObject, V TypedObject] interface {
	// ConvertFrom converts from an internal version into an external format.
	ConvertFrom(object T) (V, error)
	// ConvertTo converts from an external format into an internal version.
	ConvertTo(object V) (T, error)
}

type IdentityConverter[T TypedObject] struct{}

var _ Converter[VersionedTypedObject, VersionedTypedObject] = (*IdentityConverter[VersionedTypedObject])(nil)

func (_ IdentityConverter[T]) ConvertFrom(object T) (T, error) {
	return object, nil
}

func (_ IdentityConverter[T]) ConvertTo(object T) (T, error) {
	return object, nil
}

////////////////////////////////////////////////////////////////////////////////

type (
	FormatVersion[T VersionedTypedObject] interface {
		TypedObjectDecoder[T]
		TypedObjectEncoder[T]
	}
	// _FormatVersion[T VersionedTypedObject] = FormatVersion[T] // I like Go.
	_FormatVersion[T VersionedTypedObject] interface {
		FormatVersion[T]
	}
)

type formatVersion[T VersionedTypedObject, I VersionedTypedObject, V TypedObject] struct {
	decoder   TypedObjectDecoder[V]
	converter Converter[I, V]
}

func (c *formatVersion[T, I, V]) Encode(object T, marshaler Marshaler) ([]byte, error) {
	i, err := generics.Cast[I](object)
	if err != nil {
		return nil, err
	}
	v, err := c.converter.ConvertFrom(i)
	if err != nil {
		return nil, err
	}
	if marshaler == nil {
		marshaler = DefaultJSONEncoding
	}
	return marshaler.Marshal(v)
}

func (c *formatVersion[T, I, V]) Decode(data []byte, unmarshaler Unmarshaler) (T, error) {
	var _nil T
	v, err := c.decoder.Decode(data, unmarshaler)
	if err != nil {
		return _nil, err
	}
	i, err := c.converter.ConvertTo(v)
	if err != nil {
		return _nil, err
	}
	return generics.Cast[T](i)
}

// caster applies an implementation to interface upcast for a format version,
// here it has to be a subtype of T, but thanks to Go this cannot be expressed.
type caster[T VersionedTypedObject, I VersionedTypedObject] struct {
	version FormatVersion[I]
}

func (c *caster[T, I]) Decode(data []byte, unmarshaler Unmarshaler) (T, error) {
	var _nil T
	o, err := c.version.Decode(data, unmarshaler)
	if err != nil {
		return _nil, err
	}
	var i interface{} = o // type parameter based casts not supported by go
	if t, ok := i.(T); ok {
		return t, nil
	}
	return _nil, errors.ErrInvalid("type", fmt.Sprintf("%T", o))
}

func (c *caster[T, I]) Encode(o T, marshaler Marshaler) ([]byte, error) {
	var t interface{} = o // type parameter based casts not supported by go
	if i, ok := t.(I); ok {
		return c.version.Encode(i, marshaler)
	}
	return nil, errors.ErrInvalid("type", fmt.Sprintf("%T", o))
}

type implementation struct {
	VersionedTypedObject
}

var _ FormatVersion[VersionedTypedObject] = (*caster[VersionedTypedObject, implementation])(nil)

// NewSimpleVersion creates a new format version for versioned typed objects,
// where T is the common *interface* of all types of the same type realm.
// It creates an external version identical to the internal representation (type I).
// This must be a struct pointer type.
func NewSimpleVersion[T VersionedTypedObject, I VersionedTypedObject]() FormatVersion[T] {
	var proto I // first time use of typed structure nil pointers

	_, err := generics.Cast[T](proto)
	if err != nil {
		var t *T
		panic(fmt.Errorf("invalid type %T: does not implement required interface %s", proto, reflect.TypeOf(t).Elem()))
	}
	return &formatVersion[T, I, I]{
		decoder:   MustNewDirectDecoder[I](proto),
		converter: &IdentityConverter[I]{},
	}
}

// NewConvertedVersion creates a new format version for versioned typed objects,
// where T is the common *interface* of all types of the same type realm and I is the
// *internal implementation* commonly used for the various version variants of a dedicated kind of type,
// representing the format this format version is responsible for.
// Therefore, I must be subtype of T, which cannot be expressed in Go.
// The converter must convert between the external version, specified by the given prototype and
// the *internal* representation (type I) used to internally represent a set of variants as Go object.
func NewConvertedVersion[T VersionedTypedObject, I VersionedTypedObject, V TypedObject](converter Converter[I, V]) FormatVersion[T] {
	var proto V
	return &formatVersion[T, I, V]{
		decoder:   MustNewDirectDecoder[V](proto),
		converter: converter,
	}
}

func NewConvertedVersionByProto[T VersionedTypedObject, I VersionedTypedObject, V TypedObject](proto V, converter Converter[I, V]) FormatVersion[T] {
	return &formatVersion[T, I, V]{
		decoder:   MustNewDirectDecoder[V](proto),
		converter: converter,
	}
}
