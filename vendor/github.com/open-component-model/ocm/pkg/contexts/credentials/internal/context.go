// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"reflect"

	"github.com/open-component-model/ocm/pkg/contexts/config"
	cfgcpi "github.com/open-component-model/ocm/pkg/contexts/config/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/finalizer"
	"github.com/open-component-model/ocm/pkg/runtime"
)

// CONTEXT_TYPE is the global type for a credential context.
const CONTEXT_TYPE = "credentials" + datacontext.OCM_CONTEXT_SUFFIX

// ProviderIdentity is used to uniquely identify a provider
// for a configured consumer id. If non-empty it
// must start with a DNSname identifying the origin of the
// provider followed by a slash and a local arbitrary identity.
type ProviderIdentity = finalizer.ObjectIdentity

type ContextProvider interface {
	CredentialsContext() Context
}

type ConsumerProvider interface {
	Unregister(id ProviderIdentity)
	Get(id ConsumerIdentity) (CredentialsSource, bool)
	Match(id ConsumerIdentity, cur ConsumerIdentity, matcher IdentityMatcher) (CredentialsSource, ConsumerIdentity)
}

type Context interface {
	datacontext.Context
	ContextProvider
	config.ContextProvider

	AttributesContext() datacontext.AttributesContext
	RepositoryTypes() RepositoryTypeScheme

	RepositorySpecForConfig(data []byte, unmarshaler runtime.Unmarshaler) (RepositorySpec, error)

	RepositoryForSpec(spec RepositorySpec, creds ...CredentialsSource) (Repository, error)
	RepositoryForConfig(data []byte, unmarshaler runtime.Unmarshaler, creds ...CredentialsSource) (Repository, error)

	CredentialsForSpec(spec CredentialsSpec, creds ...CredentialsSource) (Credentials, error)
	CredentialsForConfig(data []byte, unmarshaler runtime.Unmarshaler, cred ...CredentialsSource) (Credentials, error)

	RegisterConsumerProvider(id ProviderIdentity, provider ConsumerProvider)
	UnregisterConsumerProvider(id ProviderIdentity)

	GetCredentialsForConsumer(ConsumerIdentity, ...IdentityMatcher) (CredentialsSource, error)
	SetCredentialsForConsumer(identity ConsumerIdentity, creds CredentialsSource)
	SetCredentialsForConsumerWithProvider(pid ProviderIdentity, identity ConsumerIdentity, creds CredentialsSource)

	SetAlias(name string, spec RepositorySpec, creds ...CredentialsSource) error

	ConsumerIdentityMatchers() IdentityMatcherRegistry
}

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
	return p.CredentialsContext()
}

func DefinedForContext(ctx context.Context) (Context, bool) {
	c, ok := datacontext.ForContextByKey(ctx, key, DefaultContext)
	if c != nil {
		return c.(Context), ok
	}
	return nil, ok
}

type _context struct {
	datacontext.InternalContext

	sharedattributes         datacontext.AttributesContext
	updater                  cfgcpi.Updater
	knownRepositoryTypes     RepositoryTypeScheme
	consumerIdentityMatchers IdentityMatcherRegistry
	consumerProviders        *consumerProviderRegistry
}

var _ Context = &_context{}

func newContext(configctx config.Context, reposcheme RepositoryTypeScheme, consumerMatchers IdentityMatcherRegistry, delegates datacontext.Delegates) Context {
	c := &_context{
		sharedattributes:         configctx.AttributesContext(),
		knownRepositoryTypes:     reposcheme,
		consumerIdentityMatchers: consumerMatchers,
		consumerProviders:        newConsumerProviderRegistry(),
	}
	c.InternalContext = datacontext.NewContextBase(c, CONTEXT_TYPE, key, configctx.GetAttributes(), delegates)
	c.updater = cfgcpi.NewUpdater(configctx, c)
	return c
}

func (c *_context) CredentialsContext() Context {
	return c
}

func (c *_context) Update() error {
	return c.updater.Update()
}

func (c *_context) GetType() string {
	return CONTEXT_TYPE
}

func (c *_context) AttributesContext() datacontext.AttributesContext {
	return c.sharedattributes
}

func (c *_context) ConfigContext() config.Context {
	return c.updater.GetContext()
}

func (c *_context) RepositoryTypes() RepositoryTypeScheme {
	return c.knownRepositoryTypes
}

func (c *_context) RepositorySpecForConfig(data []byte, unmarshaler runtime.Unmarshaler) (RepositorySpec, error) {
	return c.knownRepositoryTypes.Decode(data, unmarshaler)
}

func (c *_context) RepositoryForSpec(spec RepositorySpec, creds ...CredentialsSource) (Repository, error) {
	cred, err := CredentialsChain(creds).Credentials(c)
	if err != nil {
		return nil, err
	}
	c.Update()
	return spec.Repository(c, cred)
}

func (c *_context) RepositoryForConfig(data []byte, unmarshaler runtime.Unmarshaler, creds ...CredentialsSource) (Repository, error) {
	spec, err := c.knownRepositoryTypes.Decode(data, unmarshaler)
	if err != nil {
		return nil, err
	}
	return c.RepositoryForSpec(spec, creds...)
}

func (c *_context) CredentialsForSpec(spec CredentialsSpec, creds ...CredentialsSource) (Credentials, error) {
	repospec := spec.GetRepositorySpec(c)
	repo, err := c.RepositoryForSpec(repospec, creds...)
	if err != nil {
		return nil, err
	}
	return repo.LookupCredentials(spec.GetCredentialsName())
}

func (c *_context) CredentialsForConfig(data []byte, unmarshaler runtime.Unmarshaler, creds ...CredentialsSource) (Credentials, error) {
	spec := &GenericCredentialsSpec{}
	err := unmarshaler.Unmarshal(data, spec)
	if err != nil {
		return nil, err
	}
	return c.CredentialsForSpec(spec, creds...)
}

var emptyIdentity = ConsumerIdentity{}

func (c *_context) GetCredentialsForConsumer(identity ConsumerIdentity, matchers ...IdentityMatcher) (CredentialsSource, error) {
	err := c.Update()
	if err != nil {
		return nil, err
	}

	m := c.defaultMatcher(identity, matchers...)
	var credsrc CredentialsSource
	if m == nil {
		credsrc, _ = c.consumerProviders.Get(identity)
	} else {
		credsrc, _ = c.consumerProviders.Match(identity, nil, m)
	}
	if credsrc == nil {
		credsrc, _ = c.consumerProviders.Get(emptyIdentity)
	}
	if credsrc == nil {
		return nil, ErrUnknownConsumer(identity.String())
	}
	return credsrc, nil
}

func (c *_context) defaultMatcher(id ConsumerIdentity, matchers ...IdentityMatcher) IdentityMatcher {
	def := c.consumerIdentityMatchers.Get(id.Type())
	if def == nil {
		def = PartialMatch
	}
	return mergeMatcher(def, andMatcher, matchers)
}

func (c *_context) SetCredentialsForConsumer(identity ConsumerIdentity, creds CredentialsSource) {
	c.Update()
	c.consumerProviders.Set(identity, "", creds)
}

func (c *_context) SetCredentialsForConsumerWithProvider(pid ProviderIdentity, identity ConsumerIdentity, creds CredentialsSource) {
	c.Update()
	c.consumerProviders.Set(identity, pid, creds)
}

func (c *_context) ConsumerIdentityMatchers() IdentityMatcherRegistry {
	return c.consumerIdentityMatchers
}

func (c *_context) SetAlias(name string, spec RepositorySpec, creds ...CredentialsSource) error {
	c.Update()
	t := c.knownRepositoryTypes.GetType(AliasRepositoryType)
	if t == nil {
		return errors.ErrNotSupported("aliases")
	}
	if a, ok := t.(AliasRegistry); ok {
		return a.SetAlias(c, name, spec, CredentialsChain(creds))
	}
	return errors.ErrNotImplemented("interface", "AliasRegistry", reflect.TypeOf(t).String())
}

func (c *_context) RegisterConsumerProvider(id ProviderIdentity, provider ConsumerProvider) {
	c.consumerProviders.Register(id, provider)
}

func (c *_context) UnregisterConsumerProvider(id ProviderIdentity) {
	c.consumerProviders.Unregister(id)
}
