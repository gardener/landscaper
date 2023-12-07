// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package generics

import (
	"fmt"
	"reflect"

	"github.com/open-component-model/ocm/pkg/errors"
)

// TypeOf returns the reflect.Type object for a formal Go type
// given by type parameter.
func TypeOf[T any]() reflect.Type {
	var ifce T
	return reflect.TypeOf(&ifce).Elem()
}

// As casts an element typed by a type parameter
// to a subtype, given by the type parameter T.
func As[T any](o interface{}) T {
	var _nil T
	if o == nil {
		return _nil
	}
	return o.(T)
}

// AsE casts a pointer/error result to an interface/error
// result.
// In Go this cannot be done directly, because returning a nil pinter
// for an interface return type, would result is a typed nil value for
// the interface, and not nil, if the pointer is nil.
// Unfortunately, the relation of the pointer (even the fact, that a pointer is
// expected)to the interface (even the fact, that an interface is expected)
// cannot be expressed with Go generics.
func AsE[T any](o interface{}, err error) (T, error) {
	var _nil T
	if o == nil {
		return _nil, err
	}
	return o.(T), err
}

// AsI converts a struct pointer to an appropriate interface type
// given as type parameter.
// In Go this cannot be done directly, because returning a nil pinter
// for an interface return type, would result is a typed nil value for
// the interface, and not nil, if the pointer is nil.
// Unfortunately, the relation of the pointer (even the fact, that a pointer is
// expected)to the interface (even the fact, that an interface is expected)
// cannot be expressed with Go generics.
func AsI[T any](p any) T {
	var _nil T
	if p == nil {
		return _nil
	}
	return p.(T)
}

// Cast casts one type parameter to another type parameter,
// which have a sub type relation.
// This cannot be described by type parameter constraints in Go, because
// constraints may not be type parameters again.
func Cast[S any](o any) (S, error) {
	var _nil S
	var t any = o
	if s, ok := t.(S); ok {
		return s, nil
	}
	return _nil, errors.ErrInvalid("type", fmt.Sprintf("%T", o))
}

// TryCast tries a type cast usable for variable with a parametric type.
func TryCast[T any](p any) (T, bool) {
	var _nil T
	if p == nil {
		return _nil, false
	}
	t, ok := p.(T)
	return t, ok
}
