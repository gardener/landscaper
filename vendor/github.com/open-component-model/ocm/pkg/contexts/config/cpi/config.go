// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"reflect"
	"strings"

	"github.com/open-component-model/ocm/pkg/runtime"
)

type DefaultConfigType struct {
	runtime.ObjectVersionedType
	runtime.TypedObjectDecoder
	usage string
}

func NewConfigType(name string, proto Config, usages ...string) ConfigType {
	t := reflect.TypeOf(proto)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return &DefaultConfigType{
		ObjectVersionedType: runtime.NewVersionedObjectType(name),
		TypedObjectDecoder:  runtime.MustNewDirectDecoder(proto),
		usage:               strings.Join(usages, "\n"),
	}
}

func (t *DefaultConfigType) Usage() string {
	return t.usage
}
