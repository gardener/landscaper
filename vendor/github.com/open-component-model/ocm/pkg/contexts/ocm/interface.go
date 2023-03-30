// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"context"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/internal"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	KIND_COMPONENTVERSION   = internal.KIND_COMPONENTVERSION
	KIND_COMPONENTREFERENCE = "component reference"
	KIND_RESOURCE           = internal.KIND_RESOURCE
	KIND_SOURCE             = internal.KIND_SOURCE
	KIND_REFERENCE          = internal.KIND_REFERENCE
)

const CONTEXT_TYPE = internal.CONTEXT_TYPE

const CommonTransportFormat = internal.CommonTransportFormat

type (
	Context                          = internal.Context
	ContextProvider                  = internal.ContextProvider
	ComponentVersionResolver         = internal.ComponentVersionResolver
	Repository                       = internal.Repository
	RepositorySpecHandlers           = internal.RepositorySpecHandlers
	RepositorySpecHandler            = internal.RepositorySpecHandler
	UniformRepositorySpec            = internal.UniformRepositorySpec
	ComponentLister                  = internal.ComponentLister
	ComponentAccess                  = internal.ComponentAccess
	ComponentVersionAccess           = internal.ComponentVersionAccess
	AccessSpec                       = internal.AccessSpec
	HintProvider                     = internal.HintProvider
	AccessMethod                     = internal.AccessMethod
	AccessType                       = internal.AccessType
	DataAccess                       = internal.DataAccess
	BlobAccess                       = internal.BlobAccess
	SourceAccess                     = internal.SourceAccess
	SourceMeta                       = internal.SourceMeta
	ResourceAccess                   = internal.ResourceAccess
	ResourceMeta                     = internal.ResourceMeta
	RepositorySpec                   = internal.RepositorySpec
	IntermediateRepositorySpecAspect = internal.IntermediateRepositorySpecAspect
	RepositoryType                   = internal.RepositoryType
	RepositoryTypeScheme             = internal.RepositoryTypeScheme
	AccessTypeScheme                 = internal.AccessTypeScheme
	ComponentReference               = internal.ComponentReference
)

type (
	DigesterType         = internal.DigesterType
	BlobDigester         = internal.BlobDigester
	BlobDigesterRegistry = internal.BlobDigesterRegistry
	DigestDescriptor     = internal.DigestDescriptor
)

type (
	BlobHandlerRegistry = internal.BlobHandlerRegistry
	BlobHandler         = internal.BlobHandler
)

func NewDigestDescriptor(digest, hashAlgo, normAlgo string) *DigestDescriptor {
	return internal.NewDigestDescriptor(digest, hashAlgo, normAlgo)
}

// DefaultContext is the default context initialized by init functions.
func DefaultContext() internal.Context {
	return internal.DefaultContext
}

func DefaultBlobHandlers() internal.BlobHandlerRegistry {
	return internal.DefaultBlobHandlerRegistry
}

// ForContext returns the Context to use for context.Context.
// This is either an explicit context or the default context.
func ForContext(ctx context.Context) Context {
	return internal.ForContext(ctx)
}

func DefinedForContext(ctx context.Context) (Context, bool) {
	return internal.DefinedForContext(ctx)
}

func NewGenericAccessSpec(spec string) (AccessSpec, error) {
	return internal.NewGenericAccessSpec(spec)
}

type AccessSpecRef = internal.AccessSpecRef

func NewAccessSpecRef(spec cpi.AccessSpec) *AccessSpecRef {
	return internal.NewAccessSpecRef(spec)
}

func NewRawAccessSpecRef(data []byte, unmarshaler runtime.Unmarshaler) (*AccessSpecRef, error) {
	return internal.NewRawAccessSpecRef(data, unmarshaler)
}
