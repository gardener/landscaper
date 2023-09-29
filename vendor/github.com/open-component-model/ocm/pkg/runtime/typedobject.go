// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package runtime

// ObjectTypedObject is the minimal implementation of a typed object
// managing the type information.
type ObjectTypedObject = ObjectType

func NewTypedObject(typ string) ObjectTypedObject {
	return NewObjectType(typ)
}

// TypedObject defines the common interface for all kinds of typed objects.
type TypedObject interface {
	TypeInfo
}

// TypedObjectType is the interface for a type object for an TypedObject.
type TypedObjectType[T TypedObject] interface {
	TypeInfo
	TypedObjectDecoder[T]
}

type typeObject[T TypedObject] struct {
	_ObjectType
	_TypedObjectDecoder[T]
}

var _ TypedObjectType[TypedObject] = (*typeObject[TypedObject])(nil)

func NewTypedObjectTypeByDecoder[T TypedObject](name string, decoder TypedObjectDecoder[T]) TypedObjectType[T] {
	return &typeObject[T]{
		_ObjectType:         NewObjectType(name),
		_TypedObjectDecoder: decoder,
	}
}

func NewTypedObjectTypeByProto[T TypedObject](name string, proto T) TypedObjectType[T] {
	return &typeObject[T]{
		_ObjectType:         NewObjectType(name),
		_TypedObjectDecoder: MustNewDirectDecoder[T](proto),
	}
}
