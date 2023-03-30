// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"reflect"

	"github.com/open-component-model/ocm/pkg/runtime"
)

type DefaultRepositoryType struct {
	runtime.ObjectVersionedType
	runtime.TypedObjectDecoder
}

func NewRepositoryType(name string, proto RepositorySpec) RepositoryType {
	t := reflect.TypeOf(proto)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return &DefaultRepositoryType{
		ObjectVersionedType: runtime.NewVersionedObjectType(name),
		TypedObjectDecoder:  runtime.MustNewDirectDecoder(proto),
	}
}
