// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"github.com/open-component-model/ocm/pkg/cobrautils/flagsets"
	"github.com/open-component-model/ocm/pkg/runtime"
)

type AccessTypeVersionScheme = runtime.TypeVersionScheme[AccessSpec, AccessType]

func NewAccessTypeVersionScheme(kind string) AccessTypeVersionScheme {
	return runtime.NewTypeVersionScheme[AccessSpec, AccessType](kind, newStrictAccessTypeScheme())
}

func RegisterAccessType(atype AccessType) {
	defaultAccessTypeScheme.Register(atype)
}

func RegisterAccessTypeVersions(s AccessTypeVersionScheme) {
	defaultAccessTypeScheme.AddKnownTypes(s)
}

////////////////////////////////////////////////////////////////////////////////

type AccessSpecFormatVersionRegistry = runtime.FormatVersionRegistry[AccessSpec]

func NewAccessSpecFormatVersionRegistry() AccessSpecFormatVersionRegistry {
	return runtime.NewFormatVersionRegistry[AccessSpec]()
}

func MustNewAccessSpecMultiFormatVersion(kind string, formats AccessSpecFormatVersionRegistry) runtime.FormatVersion[AccessSpec] {
	return runtime.MustNewMultiFormatVersion[AccessSpec](kind, formats)
}

////////////////////////////////////////////////////////////////////////////////

type additionalTypeInfo interface {
	ConfigOptionTypeSetHandler() flagsets.ConfigOptionTypeSetHandler
	Description() string
	Format() string
}

type accessType struct {
	runtime.VersionedTypedObjectType[AccessSpec]
	description string
	format      string
	handler     flagsets.ConfigOptionTypeSetHandler
}

var _ additionalTypeInfo = (*accessType)(nil)

func NewAccessSpecTypeByBaseType(vt runtime.VersionedTypedObjectType[AccessSpec], opts ...AccessSpecTypeOption) AccessType {
	t := accessTypeTarget{&accessType{
		VersionedTypedObjectType: vt,
	}}
	for _, o := range opts {
		o.ApplyToAccessSpecOptionTarget(t)
	}
	return t.accessType
}

func NewAccessSpecType[I AccessSpec](name string, opts ...AccessSpecTypeOption) AccessType {
	return NewAccessSpecTypeByBaseType(runtime.NewVersionedTypedObjectType[AccessSpec, I](name), opts...)
}

func NewAccessSpecTypeByConverter[I AccessSpec, V runtime.VersionedTypedObject](name string, converter runtime.Converter[I, V], opts ...AccessSpecTypeOption) AccessType {
	return NewAccessSpecTypeByBaseType(runtime.NewVersionedTypedObjectTypeByConverter[AccessSpec, I, V](name, converter), opts...)
}

func NewAccessSpecTypeByFormatVersion(name string, fmt runtime.FormatVersion[AccessSpec], opts ...AccessSpecTypeOption) AccessType {
	return NewAccessSpecTypeByBaseType(runtime.NewVersionedTypedObjectTypeByFormatVersion[AccessSpec](name, fmt), opts...)
}

func (t *accessType) ConfigOptionTypeSetHandler() flagsets.ConfigOptionTypeSetHandler {
	return t.handler
}

func (t *accessType) Description() string {
	return t.description
}

func (t *accessType) Format() string {
	return t.format
}

////////////////////////////////////////////////////////////////////////////////

// accessTypeTarget is used as target for option functions, it provides
// setters for fields, which should nor be modifiable for a final type object.
type accessTypeTarget struct {
	*accessType
}

func (t accessTypeTarget) SetDescription(value string) {
	t.description = value
}

func (t accessTypeTarget) SetFormat(value string) {
	t.format = value
}

func (t accessTypeTarget) SetConfigHandler(value flagsets.ConfigOptionTypeSetHandler) {
	t.handler = value
}
