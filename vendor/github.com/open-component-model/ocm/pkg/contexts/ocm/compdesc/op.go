// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package compdesc

import (
	"reflect"

	"github.com/open-component-model/ocm/pkg/generics"
)

// unfortunately this does not work as expected in Go, because result parameters
// are not used for type inference. Therefore, these methods cannot be used
// without specifying the type parameter.

func And[T any](sel ...any) T {
	var r T
	h := selhandler[reflect.TypeOf(r)].(handler[T])
	return h.And(generics.ConvertSliceTo[T](sel)...)
}

func Or[T any](sel ...any) T {
	var r T
	h := selhandler[reflect.TypeOf(r)].(handler[T])
	return h.Or(generics.ConvertSliceTo[T](sel)...)
}

func Not[T any](sel any) T {
	var r T
	h := selhandler[reflect.TypeOf(r)].(handler[T])
	return h.Not(sel.(T))
}

////////////////////////////////////////////////////////////////////////////////

type handler[T any] interface {
	And(sel ...T) T
	Or(sel ...T) T
	Not(sel T) T
}

var (
	l_type LabelSelector
	r_type ResourceSelector
	c_type ReferenceSelector
)

var selhandler = map[reflect.Type]any{
	reflect.TypeOf(l_type): l_handler{},
	reflect.TypeOf(r_type): r_handler{},
	reflect.TypeOf(c_type): c_handler{},
}

type l_handler struct{}

func (h l_handler) And(sel ...LabelSelector) LabelSelector {
	return AndL(sel...)
}

func (h l_handler) Or(sel ...LabelSelector) LabelSelector {
	return OrL(sel...)
}

func (h l_handler) Not(sel LabelSelector) LabelSelector {
	return NotL(sel)
}

type r_handler struct{}

func (h r_handler) And(sel ...ResourceSelector) ResourceSelector {
	return AndR(sel...)
}

func (h r_handler) Or(sel ...ResourceSelector) ResourceSelector {
	return OrR(sel...)
}

func (h r_handler) Not(sel ResourceSelector) ResourceSelector {
	return NotR(sel)
}

type c_handler struct{}

func (h c_handler) And(sel ...ReferenceSelector) ReferenceSelector {
	return AndC(sel...)
}

func (h c_handler) Or(sel ...ReferenceSelector) ReferenceSelector {
	return OrC(sel...)
}

func (h c_handler) Not(sel ReferenceSelector) ReferenceSelector {
	return NotC(sel)
}
