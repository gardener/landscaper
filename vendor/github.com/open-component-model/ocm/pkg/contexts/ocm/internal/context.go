// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"reflect"
	"strings"

	. "github.com/open-component-model/ocm/pkg/finalizer"

	"github.com/modern-go/reflect2"

	"github.com/open-component-model/ocm/pkg/contexts/config"
	cfgcpi "github.com/open-component-model/ocm/pkg/contexts/config/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/ctf"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const CONTEXT_TYPE = "ocm" + datacontext.OCM_CONTEXT_SUFFIX

const CommonTransportFormat = ctf.Type

type ContextProvider interface {
	OCMContext() Context
}

type LocalContextProvider interface {
	GetContext() Context
}

type localContextProvider struct {
	LocalContextProvider
}

func (l *localContextProvider) OCMContext() Context {
	return l.GetContext()
}

func WrapContextProvider(ctx LocalContextProvider) ContextProvider {
	return &localContextProvider{ctx}
}

type Context interface {
	datacontext.Context
	config.ContextProvider
	credentials.ContextProvider
	oci.ContextProvider
	ContextProvider

	RepositoryTypes() RepositoryTypeScheme
	AccessMethods() AccessTypeScheme

	RepositorySpecHandlers() RepositorySpecHandlers
	MapUniformRepositorySpec(u *UniformRepositorySpec) (RepositorySpec, error)

	DisableBlobHandlers()
	BlobHandlers() BlobHandlerRegistry
	BlobDigesters() BlobDigesterRegistry

	RepositoryForSpec(spec RepositorySpec, creds ...credentials.CredentialsSource) (Repository, error)
	RepositoryForConfig(data []byte, unmarshaler runtime.Unmarshaler, creds ...credentials.CredentialsSource) (Repository, error)
	RepositorySpecForConfig(data []byte, unmarshaler runtime.Unmarshaler) (RepositorySpec, error)

	AccessSpecForSpec(spec compdesc.AccessSpec) (AccessSpec, error)
	AccessSpecForConfig(data []byte, unmarshaler runtime.Unmarshaler) (AccessSpec, error)

	Encode(AccessSpec, runtime.Marshaler) ([]byte, error)

	GetAlias(name string) RepositorySpec
	SetAlias(name string, spec RepositorySpec)

	GetResolver() ComponentVersionResolver
	AddResolverRule(prefix string, spec RepositorySpec, prio ...int)

	// Finalize finalizes elements implicitly opened during resource operations.
	// For example, registered blob handler may open objects, which are kept open
	// for performance reasons. At the end of a usage finalize should be called
	// to finalize those elements. This method can be called any time by a context
	// user to cleanup temporarily allocated resources. Therefore, only
	// elements should be added to the finalzer, which can be reopened/created
	// on-the fly whenever required.
	Finalize() error
	Finalizer() *Finalizer
}

// //////////////////////////////////////////////////////////////////////////////

var key = reflect.TypeOf(_context{})

// DefaultContext is the default context initialized by init functions.
var DefaultContext = Builder{}.New(datacontext.MODE_SHARED)

// FromContext returns the Context to use for context.Context.
// This is either an explicit context or the default context.
func FromContext(ctx context.Context) Context {
	c, _ := datacontext.ForContextByKey(ctx, key, DefaultContext)
	return c.(Context)
}

func FromProvider(p ContextProvider) Context {
	if p == nil {
		return nil
	}
	return p.OCMContext()
}

func DefinedForContext(ctx context.Context) (Context, bool) {
	c, ok := datacontext.ForContextByKey(ctx, key, DefaultContext)
	if c != nil {
		return c.(Context), ok
	}
	return nil, ok
}

// //////////////////////////////////////////////////////////////////////////////

type _InternalContext = datacontext.InternalContext

type _context struct {
	_InternalContext
	updater cfgcpi.Updater

	sharedattributes datacontext.AttributesContext
	credctx          credentials.Context
	ocictx           oci.Context

	knownRepositoryTypes RepositoryTypeScheme
	knownAccessTypes     AccessTypeScheme

	specHandlers  RepositorySpecHandlers
	blobHandlers  BlobHandlerRegistry
	blobDigesters BlobDigesterRegistry
	aliases       map[string]RepositorySpec
	resolver      *resolver
}

var _ Context = &_context{}

// gcWrapper is used as garbage collectable
// wrapper for a context implementation
// to establish a runtime finalizer.
type gcWrapper struct {
	datacontext.GCWrapper
	*_context
}

func (w *gcWrapper) SetContext(c *_context) {
	w._context = c
}

func newContext(credctx credentials.Context, ocictx oci.Context, reposcheme RepositoryTypeScheme, accessscheme AccessTypeScheme, specHandlers RepositorySpecHandlers, blobHandlers BlobHandlerRegistry, blobDigesters BlobDigesterRegistry, repodel RepositoryDelegationRegistry, delegates datacontext.Delegates) Context {
	c := &_context{
		sharedattributes:     credctx.AttributesContext(),
		credctx:              credctx,
		ocictx:               ocictx,
		specHandlers:         specHandlers,
		blobHandlers:         blobHandlers,
		blobDigesters:        blobDigesters,
		knownAccessTypes:     accessscheme,
		knownRepositoryTypes: reposcheme,
		aliases:              map[string]RepositorySpec{},
	}

	if repodel != nil {
		c.knownRepositoryTypes = NewRepositoryTypeScheme(&delegatingDecoder{ctx: c, delegate: repodel}, reposcheme)
	}
	c._InternalContext = datacontext.NewContextBase(c, CONTEXT_TYPE, key, credctx.GetAttributes(), delegates)
	c.updater = cfgcpi.NewUpdater(credctx.ConfigContext(), c)
	c.resolver = &resolver{
		ctx:              c,
		MatchingResolver: NewMatchingResolver(c),
	}
	c.Finalizer().With(c.resolver.Finalize)
	return datacontext.FinalizedContext[gcWrapper](c)
}

func (c *_context) OCMContext() Context {
	return c
}

func (c *_context) Update() error {
	return c.updater.Update()
}

func (c *_context) AttributesContext() datacontext.AttributesContext {
	return c.sharedattributes
}

func (c *_context) ConfigContext() config.Context {
	return c.updater.GetContext()
}

func (c *_context) CredentialsContext() credentials.Context {
	return c.credctx
}

func (c *_context) OCIContext() oci.Context {
	return c.ocictx
}

func (c *_context) RepositoryTypes() RepositoryTypeScheme {
	return c.knownRepositoryTypes
}

func (c *_context) RepositorySpecHandlers() RepositorySpecHandlers {
	return c.specHandlers
}

func (c *_context) MapUniformRepositorySpec(u *UniformRepositorySpec) (RepositorySpec, error) {
	return c.specHandlers.MapUniformRepositorySpec(c, u)
}

func (c *_context) DisableBlobHandlers() {
	c.blobHandlers = NewBlobHandlerRegistry(nil)
}

func (c *_context) BlobHandlers() BlobHandlerRegistry {
	c.Update()
	return c.blobHandlers
}

func (c *_context) BlobDigesters() BlobDigesterRegistry {
	c.Update()
	return c.blobDigesters
}

func (c *_context) RepositoryForSpec(spec RepositorySpec, creds ...credentials.CredentialsSource) (Repository, error) {
	cred, err := credentials.CredentialsChain(creds).Credentials(c.CredentialsContext())
	if err != nil {
		return nil, err
	}
	return spec.Repository(c, cred)
}

func (c *_context) RepositoryForConfig(data []byte, unmarshaler runtime.Unmarshaler, creds ...credentials.CredentialsSource) (Repository, error) {
	spec, err := c.knownRepositoryTypes.Decode(data, unmarshaler)
	if err != nil {
		return nil, err
	}
	return c.RepositoryForSpec(spec, creds...)
}

func (c *_context) RepositorySpecForConfig(data []byte, unmarshaler runtime.Unmarshaler) (RepositorySpec, error) {
	return c.knownRepositoryTypes.Decode(data, unmarshaler)
}

func (c *_context) AccessMethods() AccessTypeScheme {
	return c.knownAccessTypes
}

func (c *_context) AccessSpecForConfig(data []byte, unmarshaler runtime.Unmarshaler) (AccessSpec, error) {
	return c.knownAccessTypes.Decode(data, unmarshaler)
}

// AccessSpecForSpec takes an anonymous access specification and tries to map
// it to an effective implementation.
func (c *_context) AccessSpecForSpec(spec compdesc.AccessSpec) (AccessSpec, error) {
	if reflect2.IsNil(spec) {
		return nil, nil
	}
	if n, ok := spec.(AccessSpec); ok {
		if g, ok := spec.(EvaluatableAccessSpec); ok {
			return g.Evaluate(c)
		}
		return n, nil
	}
	un, err := runtime.ToUnstructuredTypedObject(spec)
	if err != nil {
		return nil, err
	}

	raw, err := un.GetRaw()
	if err != nil {
		return nil, err
	}

	return c.knownAccessTypes.Decode(raw, runtime.DefaultJSONEncoding)
}

func (c *_context) Encode(spec AccessSpec, marshaler runtime.Marshaler) ([]byte, error) {
	return c.knownAccessTypes.Encode(spec, marshaler)
}

func (c *_context) GetAlias(name string) RepositorySpec {
	err := c.updater.Update()
	if err != nil {
		return nil
	}
	c.updater.RLock()
	defer c.updater.RUnlock()
	spec := c.aliases[name]
	if spec == nil && strings.HasSuffix(name, ".alias") {
		spec = c.aliases[name[:len(name)-6]]
	}
	return spec
}

func (c *_context) SetAlias(name string, spec RepositorySpec) {
	c.updater.Lock()
	defer c.updater.Unlock()
	c.aliases[name] = spec
}

func (c *_context) GetResolver() ComponentVersionResolver {
	c.Update()
	if len(c.resolver.rules) == 0 {
		return nil
	}
	return c.resolver
}

func (c *_context) AddResolverRule(prefix string, spec RepositorySpec, prio ...int) {
	c.resolver.AddRule(prefix, spec, prio...)
}

type resolver struct {
	ctx *_context
	*MatchingResolver
}

func (r *resolver) LookupComponentVersion(name, version string) (ComponentVersionAccess, error) {
	r.ctx.Update()
	return r.MatchingResolver.LookupComponentVersion(name, version)
}
