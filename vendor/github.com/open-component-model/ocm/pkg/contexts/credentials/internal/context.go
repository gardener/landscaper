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
	"github.com/open-component-model/ocm/pkg/runtime"
)

// CONTEXT_TYPE is the global type for a credential context.
const CONTEXT_TYPE = "credentials" + datacontext.OCM_CONTEXT_SUFFIX

type ContextProvider interface {
	CredentialsContext() Context
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

	GetCredentialsForConsumer(ConsumerIdentity, ...IdentityMatcher) (CredentialsSource, error)
	SetCredentialsForConsumer(identity ConsumerIdentity, creds CredentialsSource)

	SetAlias(name string, spec RepositorySpec, creds ...CredentialsSource) error

	ConsumerIdentityMatchers() IdentityMatcherRegistry
}

var key = reflect.TypeOf(_context{})

// DefaultContext is the default context initialized by init functions.
var DefaultContext = Builder{}.New(datacontext.MODE_SHARED)

// ForContext returns the Context to use for context.Context.
// This is either an explicit context or the default context.
func ForContext(ctx context.Context) Context {
	c, _ := datacontext.ForContextByKey(ctx, key, DefaultContext)
	return c.(Context)
}

func DefinedForContext(ctx context.Context) (Context, bool) {
	c, ok := datacontext.ForContextByKey(ctx, key, DefaultContext)
	if c != nil {
		return c.(Context), ok
	}
	return nil, ok
}

////////////////////////////////////////////////////////////////////////////////

type _context struct {
	datacontext.Context

	sharedattributes         datacontext.AttributesContext
	updater                  cfgcpi.Updater
	knownRepositoryTypes     RepositoryTypeScheme
	consumerIdentityMatchers IdentityMatcherRegistry
	consumers                *_consumers
}

var _ Context = &_context{}

func newContext(configctx config.Context, reposcheme RepositoryTypeScheme, consumerMatchers IdentityMatcherRegistry, delegates datacontext.Delegates) Context {
	c := &_context{
		sharedattributes:         configctx.AttributesContext(),
		knownRepositoryTypes:     reposcheme,
		consumerIdentityMatchers: consumerMatchers,
		consumers:                newConsumers(),
	}
	c.Context = datacontext.NewContextBase(c, CONTEXT_TYPE, key, configctx.GetAttributes(), delegates)
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
	return c.knownRepositoryTypes.DecodeRepositorySpec(data, unmarshaler)
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
	spec, err := c.knownRepositoryTypes.DecodeRepositorySpec(data, unmarshaler)
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

	m := defaultMatcher(matchers...)
	var consumer *_consumer
	if m == nil {
		consumer = c.consumers.Get(identity)
	} else {
		consumer = c.consumers.Match(identity, m)
	}
	if consumer == nil {
		consumer = c.consumers.Get(emptyIdentity)
	}
	if consumer == nil {
		return nil, ErrUnknownConsumer(identity.String())
	}
	return consumer.GetCredentials(), nil
}

func (c *_context) SetCredentialsForConsumer(identity ConsumerIdentity, creds CredentialsSource) {
	c.Update()
	c.consumers.Set(identity, creds)
}

func (c *_context) ConsumerIdentityMatchers() IdentityMatcherRegistry {
	return c.consumerIdentityMatchers
}

func (c *_context) SetAlias(name string, spec RepositorySpec, creds ...CredentialsSource) error {
	c.Update()
	t := c.knownRepositoryTypes.GetRepositoryType(AliasRepositoryType)
	if t == nil {
		return errors.ErrNotSupported("aliases")
	}
	if a, ok := t.(AliasRegistry); ok {
		return a.SetAlias(c, name, spec, CredentialsChain(creds))
	}
	return errors.ErrNotImplemented("interface", "AliasRegistry", reflect.TypeOf(t).String())
}
