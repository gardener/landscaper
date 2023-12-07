// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

// this file is identical for contexts oci and credentials and similar for
// ocm.

import (
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/runtime/descriptivetype"
)

type RepositoryTypeVersionScheme = runtime.TypeVersionScheme[RepositorySpec, RepositoryType]

func NewRepositoryTypeVersionScheme(kind string) RepositoryTypeVersionScheme {
	return runtime.NewTypeVersionScheme[RepositorySpec, RepositoryType](kind, newStrictRepositoryTypeScheme())
}

func RegisterRepositoryType(rtype RepositoryType) {
	defaultRepositoryTypeScheme.Register(rtype)
}

func RegisterRepositoryTypeVersions(s RepositoryTypeVersionScheme) {
	defaultRepositoryTypeScheme.AddKnownTypes(s)
}

////////////////////////////////////////////////////////////////////////////////

func NewRepositoryType[I RepositorySpec](name string, opts ...RepositoryOption) RepositoryType {
	return descriptivetype.NewTypedObjectTypeObject(runtime.NewVersionedTypedObjectType[RepositorySpec, I](name), opts...)
}

func NewRepositoryTypeByConverter[I RepositorySpec, V runtime.TypedObject](name string, converter runtime.Converter[I, V], opts ...RepositoryOption) RepositoryType {
	return descriptivetype.NewTypedObjectTypeObject(runtime.NewVersionedTypedObjectTypeByConverter[RepositorySpec, I](name, converter), opts...)
}

func NewRepositoryTypeByFormatVersion(name string, fmt runtime.FormatVersion[RepositorySpec], opts ...RepositoryOption) RepositoryType {
	return descriptivetype.NewTypedObjectTypeObject(runtime.NewVersionedTypedObjectTypeByFormatVersion[RepositorySpec](name, fmt), opts...)
}

////////////////////////////////////////////////////////////////////////////////

type RepositoryOption = descriptivetype.Option

func WithDescription(v string) RepositoryOption {
	return descriptivetype.WithDescription(v)
}

func WithFormatSpec(v string) RepositoryOption {
	return descriptivetype.WithFormatSpec(v)
}
