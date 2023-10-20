// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/clitypes"
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

func NewAccessSpecType[I AccessSpec](name string, opts ...AccessSpecTypeOption) AccessType {
	return clitypes.NewCLITypedObjectTypeObject[AccessSpec](runtime.NewVersionedTypedObjectType[AccessSpec, I](name), opts...)
}

func NewAccessSpecTypeByConverter[I AccessSpec, V runtime.VersionedTypedObject](name string, converter runtime.Converter[I, V], opts ...AccessSpecTypeOption) AccessType {
	return clitypes.NewCLITypedObjectTypeObject[AccessSpec](runtime.NewVersionedTypedObjectTypeByConverter[AccessSpec, I, V](name, converter), opts...)
}

func NewAccessSpecTypeByFormatVersion(name string, fmt runtime.FormatVersion[AccessSpec], opts ...AccessSpecTypeOption) AccessType {
	return clitypes.NewCLITypedObjectTypeObject[AccessSpec](runtime.NewVersionedTypedObjectTypeByFormatVersion[AccessSpec](name, fmt), opts...)
}
