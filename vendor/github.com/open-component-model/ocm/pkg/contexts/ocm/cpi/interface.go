// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

// This is the Context Provider Interface for credential providers

import (
	_ "unsafe"

	"github.com/mandelsoft/logging"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/internal"
	"github.com/open-component-model/ocm/pkg/generics"
	"github.com/open-component-model/ocm/pkg/registrations"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const CONTEXT_TYPE = internal.CONTEXT_TYPE

const CommonTransportFormat = internal.CommonTransportFormat

var TAG_BLOBHANDLER = logging.DefineTag("blobhandler", "execution of blob handler used to upload resource blobs to an ocm repository.")

func BlobHandlerLogger(ctx Context, messageContext ...logging.MessageContext) logging.Logger {
	if len(messageContext) > 0 {
		messageContext = generics.AppendedSlice[logging.MessageContext](messageContext, TAG_BLOBHANDLER)
		return ctx.Logger(messageContext...)
	} else {
		return ctx.Logger(TAG_BLOBHANDLER)
	}
}

type (
	Context                          = internal.Context
	ContextProvider                  = internal.ContextProvider
	LocalContextProvider             = internal.LocalContextProvider
	ComponentVersionResolver         = internal.ComponentVersionResolver
	Repository                       = internal.Repository
	RepositoryTypeProvider           = internal.RepositoryTypeProvider
	RepositoryTypeScheme             = internal.RepositoryTypeScheme
	RepositoryDelegationRegistry     = internal.RepositoryDelegationRegistry
	RepositoryPriorityDecoder        = internal.PriorityDecoder[Context, RepositorySpec]
	RepositorySpecHandlers           = internal.RepositorySpecHandlers
	RepositorySpecHandler            = internal.RepositorySpecHandler
	UniformRepositorySpec            = internal.UniformRepositorySpec
	ComponentLister                  = internal.ComponentLister
	ComponentAccess                  = internal.ComponentAccess
	ComponentVersionAccess           = internal.ComponentVersionAccess
	AccessSpec                       = internal.AccessSpec
	AccessSpecDecoder                = internal.AccessSpecDecoder
	GenericAccessSpec                = internal.GenericAccessSpec
	AccessMethod                     = internal.AccessMethod
	AccessMethodSupport              = internal.AccessMethodSupport
	AccessType                       = internal.AccessType
	AccessTypeProvider               = internal.AccessTypeProvider
	AccessTypeScheme                 = internal.AccessTypeScheme
	DataAccess                       = internal.DataAccess
	BlobAccess                       = internal.BlobAccess
	SourceAccess                     = internal.SourceAccess
	SourceMeta                       = internal.SourceMeta
	ResourceAccess                   = internal.ResourceAccess
	ResourceMeta                     = internal.ResourceMeta
	RepositorySpec                   = internal.RepositorySpec
	RepositorySpecDecoder            = internal.RepositorySpecDecoder
	IntermediateRepositorySpecAspect = internal.IntermediateRepositorySpecAspect
	GenericRepositorySpec            = internal.GenericRepositorySpec
	RepositoryType                   = internal.RepositoryType
	ComponentReference               = internal.ComponentReference
)

type (
	BlobHandler                  = internal.BlobHandler
	BlobHandlerOption            = internal.BlobHandlerOption
	BlobHandlerOptions           = internal.BlobHandlerOptions
	BlobHandlerKey               = internal.BlobHandlerKey
	BlobHandlerRegistry          = internal.BlobHandlerRegistry
	StorageContext               = internal.StorageContext
	ImplementationRepositoryType = internal.ImplementationRepositoryType

	BlobHandlerConfig              = internal.BlobHandlerConfig
	BlobHandlerRegistrationHandler = internal.BlobHandlerRegistrationHandler
)

type (
	DigesterType         = internal.DigesterType
	BlobDigester         = internal.BlobDigester
	BlobDigesterRegistry = internal.BlobDigesterRegistry
	DigestDescriptor     = internal.DigestDescriptor
	HasherProvider       = internal.HasherProvider
	Hasher               = internal.Hasher
)

type NamePath = registrations.NamePath

func NewNamePath(p string) NamePath {
	return registrations.NewNamePath(p)
}

func FromProvider(p ContextProvider) Context {
	return internal.FromProvider(p)
}

func NewBlobHandlerOptions(olist ...BlobHandlerOption) *BlobHandlerOptions {
	return internal.NewBlobHandlerOptions(olist...)
}

func New() Context {
	return internal.Builder{}.New()
}

func NewResourceMeta(name string, typ string, relation metav1.ResourceRelation) *ResourceMeta {
	return compdesc.NewResourceMeta(name, typ, relation)
}

func NewDigestDescriptor(digest string, typ DigesterType) *DigestDescriptor {
	return internal.NewDigestDescriptor(digest, typ.HashAlgorithm, typ.NormalizationAlgorithm)
}

func DefaultBlobDigesterRegistry() BlobDigesterRegistry {
	return internal.DefaultBlobDigesterRegistry
}

func DefaultDelegationRegistry() RepositoryDelegationRegistry {
	return internal.DefaultRepositoryDelegationRegistry
}

func DefaultContext() internal.Context {
	return internal.DefaultContext
}

func WithPrio(p int) BlobHandlerOption {
	return internal.WithPrio(p)
}

func ForRepo(ctxtype, repostype string) BlobHandlerOption {
	return internal.ForRepo(ctxtype, repostype)
}

func ForMimeType(mimetype string) BlobHandlerOption {
	return internal.ForMimeType(mimetype)
}

func ForArtifactType(arttype string) BlobHandlerOption {
	return internal.ForArtifactType(arttype)
}

func RegisterRepositorySpecHandler(handler RepositorySpecHandler, types ...string) {
	internal.RegisterRepositorySpecHandler(handler, types...)
}

func RegisterBlobHandler(handler BlobHandler, opts ...BlobHandlerOption) {
	internal.RegisterBlobHandler(handler, opts...)
}

func RegisterBlobHandlerRegistrationHandler(path string, handler BlobHandlerRegistrationHandler) {
	internal.RegisterBlobHandlerRegistrationHandler(path, handler)
}

func MustRegisterDigester(digester BlobDigester, arttypes ...string) {
	internal.MustRegisterDigester(digester, arttypes...)
}

func SetDefaultDigester(d BlobDigester) {
	internal.SetDefaultDigester(d)
}

func ToGenericAccessSpec(spec AccessSpec) (*GenericAccessSpec, error) {
	return internal.ToGenericAccessSpec(spec)
}

func ToGenericRepositorySpec(spec RepositorySpec) (*GenericRepositorySpec, error) {
	return internal.ToGenericRepositorySpec(spec)
}

type AccessSpecRef = internal.AccessSpecRef

func NewAccessSpecRef(spec AccessSpec) *AccessSpecRef {
	return internal.NewAccessSpecRef(spec)
}

func NewRawAccessSpecRef(data []byte, unmarshaler runtime.Unmarshaler) (*AccessSpecRef, error) {
	return internal.NewRawAccessSpecRef(data, unmarshaler)
}

const (
	KIND_COMPONENTVERSION = internal.KIND_COMPONENTVERSION
	KIND_RESOURCE         = internal.KIND_RESOURCE
	KIND_SOURCE           = internal.KIND_SOURCE
	KIND_REFERENCE        = internal.KIND_REFERENCE
)

func ErrComponentVersionNotFound(name, version string) error {
	return internal.ErrComponentVersionNotFound(name, version)
}

func ErrComponentVersionNotFoundWrap(err error, name, version string) error {
	return internal.ErrComponentVersionNotFoundWrap(err, name, version)
}

// PrefixProvider is supported by RepositorySpecs to
// provide info about a potential path prefix to
// use for globalized local artifacts.
type PrefixProvider interface {
	PathPrefix() string
}

func RepositoryPrefix(spec RepositorySpec) string {
	if s, ok := spec.(PrefixProvider); ok {
		return s.PathPrefix()
	}
	return ""
}

// HintProvider is able to provide a name hint for globalization of local
// artifacts.
type HintProvider internal.HintProvider

func ArtifactNameHint(spec AccessSpec, cv ComponentVersionAccess) string {
	if h, ok := spec.(HintProvider); ok {
		return h.GetReferenceHint(cv)
	}
	return ""
}

// provide context interface for other files to avoid diffs in imports.
var (
	newStrictRepositoryTypeScheme = internal.NewStrictRepositoryTypeScheme
	defaultRepositoryTypeScheme   = internal.DefaultRepositoryTypeScheme

	newStrictAccessTypeScheme = internal.NewStrictAccessTypeScheme
	defaultAccessTypeScheme   = internal.DefaultAccessTypeScheme
)

func WrapContextProvider(ctx LocalContextProvider) ContextProvider {
	return internal.WrapContextProvider(ctx)
}
