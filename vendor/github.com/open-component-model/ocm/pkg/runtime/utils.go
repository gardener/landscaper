// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"reflect"
	"sort"
	"strings"

	"github.com/open-component-model/ocm/pkg/errors"
)

func MustProtoType(proto interface{}) reflect.Type {
	t, err := ProtoType(proto)
	if err != nil {
		panic(err.Error())
	}
	return t
}

func ProtoType(proto interface{}) (reflect.Type, error) {
	if proto == nil {
		return nil, errors.New("prototype required")
	}
	t := reflect.TypeOf(proto)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, errors.Newf("prototype %q must be a struct", t)
	}
	return t, nil
}

func TypedObjectFactory(proto TypedObject) func() TypedObject {
	return func() TypedObject { return reflect.New(MustProtoType(proto)).Interface().(TypedObject) }
}

func TypeNames(scheme Scheme) []string {
	types := []string{}
	for t := range scheme.KnownTypes() {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

func KindNames(scheme Scheme) []string {
	types := []string{}
	for t := range scheme.KnownTypes() {
		if !strings.Contains(t, VersionSeparator) {
			types = append(types, t)
		}
	}
	sort.Strings(types)
	return types
}
