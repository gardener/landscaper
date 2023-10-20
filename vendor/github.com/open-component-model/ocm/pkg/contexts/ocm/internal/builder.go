// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"

	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/runtime"
)

type Builder struct {
	ctx           context.Context
	credentials   credentials.Context
	oci           oci.Context
	reposcheme    RepositoryTypeScheme
	repodel       RepositoryDelegationRegistry
	accessscheme  AccessTypeScheme
	spechandlers  RepositorySpecHandlers
	blobhandlers  BlobHandlerRegistry
	blobdigesters BlobDigesterRegistry
}

func (b *Builder) getContext() context.Context {
	if b.ctx == nil {
		return context.Background()
	}
	return b.ctx
}

func (b Builder) WithContext(ctx context.Context) Builder {
	b.ctx = ctx
	return b
}

func (b Builder) WithCredentials(ctx credentials.Context) Builder {
	b.credentials = ctx
	return b
}

func (b Builder) WithOCIRepositories(ctx oci.Context) Builder {
	b.oci = ctx
	return b
}

func (b Builder) WithRepositoyTypeScheme(scheme RepositoryTypeScheme) Builder {
	b.reposcheme = scheme
	return b
}

func (b Builder) WithRepositoryDelegation(reg RepositoryDelegationRegistry) Builder {
	b.repodel = reg
	return b
}

func (b Builder) WithAccessTypeScheme(scheme AccessTypeScheme) Builder {
	b.accessscheme = scheme
	return b
}

func (b Builder) WithRepositorySpecHandlers(reg RepositorySpecHandlers) Builder {
	b.spechandlers = reg
	return b
}

func (b Builder) WithBlobHandlers(reg BlobHandlerRegistry) Builder {
	b.blobhandlers = reg
	return b
}

func (b Builder) WithBlobDigesters(reg BlobDigesterRegistry) Builder {
	b.blobdigesters = reg
	return b
}

func (b Builder) Bound() (Context, context.Context) {
	c := b.New()
	return c, context.WithValue(b.getContext(), key, c)
}

func (b Builder) New(m ...datacontext.BuilderMode) Context {
	mode := datacontext.Mode(m...)
	ctx := b.getContext()

	if b.oci == nil {
		if b.credentials != nil {
			b.oci = oci.WithCredentials(b.credentials).New(mode)
		} else {
			var ok bool
			b.oci, ok = oci.DefinedForContext(ctx)
			if !ok && mode != datacontext.MODE_SHARED {
				b.oci = oci.New(mode)
			}
		}
	}
	if b.credentials == nil {
		b.credentials = b.oci.CredentialsContext()
	}
	if b.reposcheme == nil {
		switch mode {
		case datacontext.MODE_INITIAL:
			b.reposcheme = NewRepositoryTypeScheme(nil)
		case datacontext.MODE_CONFIGURED:
			b.reposcheme = NewRepositoryTypeScheme(nil)
			b.reposcheme.AddKnownTypes(DefaultRepositoryTypeScheme)
		case datacontext.MODE_EXTENDED:
			b.reposcheme = NewRepositoryTypeScheme(nil, DefaultRepositoryTypeScheme)
		case datacontext.MODE_DEFAULTED:
			fallthrough
		case datacontext.MODE_SHARED:
			b.reposcheme = DefaultRepositoryTypeScheme
		}
	}
	if b.accessscheme == nil {
		switch mode {
		case datacontext.MODE_INITIAL:
			b.accessscheme = NewAccessTypeScheme()
		case datacontext.MODE_CONFIGURED:
			b.accessscheme = NewAccessTypeScheme()
			b.accessscheme.AddKnownTypes(DefaultAccessTypeScheme)
		case datacontext.MODE_EXTENDED:
			b.accessscheme = NewAccessTypeScheme(DefaultAccessTypeScheme)
		case datacontext.MODE_DEFAULTED:
			fallthrough
		case datacontext.MODE_SHARED:
			b.accessscheme = DefaultAccessTypeScheme
		}
	}
	if b.spechandlers == nil {
		switch mode {
		case datacontext.MODE_INITIAL:
			b.spechandlers = NewRepositorySpecHandlers()
		case datacontext.MODE_CONFIGURED:
			b.spechandlers = DefaultRepositorySpecHandlers.Copy()
		case datacontext.MODE_EXTENDED:
			fallthrough
		case datacontext.MODE_DEFAULTED:
			fallthrough
		case datacontext.MODE_SHARED:
			b.spechandlers = DefaultRepositorySpecHandlers
		}
	}
	if b.repodel == nil {
		switch mode {
		case datacontext.MODE_INITIAL:
			b.repodel = nil
		case datacontext.MODE_CONFIGURED:
			b.repodel = DefaultRepositoryDelegationRegistry.Copy()
		case datacontext.MODE_EXTENDED:
			b.repodel = NewDelegationRegistry[Context, RepositorySpec](DefaultRepositoryDelegationRegistry)
		case datacontext.MODE_DEFAULTED:
			fallthrough
		case datacontext.MODE_SHARED:
			b.repodel = DefaultRepositoryDelegationRegistry
		}
	}
	if b.blobhandlers == nil {
		switch mode {
		case datacontext.MODE_INITIAL:
			b.blobhandlers = NewBlobHandlerRegistry()
		case datacontext.MODE_CONFIGURED:
			b.blobhandlers = DefaultBlobHandlerRegistry.Copy()
		case datacontext.MODE_EXTENDED:
			b.blobhandlers = NewBlobHandlerRegistry(DefaultBlobHandlerRegistry)
		case datacontext.MODE_DEFAULTED:
			fallthrough
		case datacontext.MODE_SHARED:
			b.blobhandlers = DefaultBlobHandlerRegistry
		}
	}
	if b.blobdigesters == nil {
		switch mode {
		case datacontext.MODE_INITIAL:
			b.blobdigesters = NewBlobDigesterRegistry()
		case datacontext.MODE_CONFIGURED:
			b.blobdigesters = DefaultBlobDigesterRegistry.Copy()
		case datacontext.MODE_EXTENDED:
			b.blobdigesters = NewBlobDigesterRegistry(DefaultBlobDigesterRegistry)
		case datacontext.MODE_DEFAULTED:
			fallthrough
		case datacontext.MODE_SHARED:
			b.blobdigesters = DefaultBlobDigesterRegistry
		}
	}

	return datacontext.SetupContext(mode, newContext(b.credentials, b.oci, b.reposcheme, b.accessscheme, b.spechandlers, b.blobhandlers, b.blobdigesters, b.repodel, b.credentials.ConfigContext()))
}

type delegatingDecoder struct {
	ctx      Context
	delegate RepositoryDelegationRegistry
}

var _ RepositorySpecDecoder = (*delegatingDecoder)(nil)

func (d *delegatingDecoder) Decode(data []byte, unmarshaler runtime.Unmarshaler) (RepositorySpec, error) {
	if d.delegate != nil {
		return d.delegate.Decode(d.ctx, data, unmarshaler)
	}
	return nil, nil
}
