// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"reflect"
	"strings"

	"github.com/open-component-model/ocm/pkg/contexts/config"
	cfgcpi "github.com/open-component-model/ocm/pkg/contexts/config/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const CONTEXT_TYPE = "oci" + datacontext.OCM_CONTEXT_SUFFIX

const CommonTransportFormat = "CommonTransportFormat"

type ContextProvider interface {
	OCIContext() Context
}

type Context interface {
	datacontext.Context
	config.ContextProvider
	credentials.ContextProvider
	ContextProvider

	RepositorySpecHandlers() RepositorySpecHandlers
	MapUniformRepositorySpec(u *UniformRepositorySpec) (RepositorySpec, error)

	RepositoryTypes() RepositoryTypeScheme

	RepositoryForSpec(spec RepositorySpec, creds ...credentials.CredentialsSource) (Repository, error)
	RepositoryForConfig(data []byte, unmarshaler runtime.Unmarshaler, creds ...credentials.CredentialsSource) (Repository, error)

	GetAlias(name string) RepositorySpec
	SetAlias(name string, spec RepositorySpec)
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

func FromProvider(p ContextProvider) Context {
	if p == nil {
		return nil
	}
	return p.OCIContext()
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
	updater cfgcpi.Updater

	sharedattributes datacontext.AttributesContext
	credentials      credentials.Context

	knownRepositoryTypes RepositoryTypeScheme
	specHandlers         RepositorySpecHandlers
	aliases              map[string]RepositorySpec
}

var _ Context = &_context{}

func newContext(credctx credentials.Context, reposcheme RepositoryTypeScheme, specHandlers RepositorySpecHandlers, delegates datacontext.Delegates) Context {
	c := &_context{
		sharedattributes:     credctx.AttributesContext(),
		credentials:          credctx,
		knownRepositoryTypes: reposcheme,
		specHandlers:         specHandlers,
		aliases:              map[string]RepositorySpec{},
	}
	c.Context = datacontext.NewContextBase(c, CONTEXT_TYPE, key, credctx.ConfigContext().GetAttributes(), delegates)
	c.updater = cfgcpi.NewUpdater(credctx.ConfigContext(), c)
	return c
}

func (c *_context) OCIContext() Context {
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
	return c.credentials
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

func (c *_context) RepositorySpecForConfig(data []byte, unmarshaler runtime.Unmarshaler) (RepositorySpec, error) {
	return c.knownRepositoryTypes.Decode(data, unmarshaler)
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
