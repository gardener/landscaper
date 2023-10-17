// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"

	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
)

type Builder struct {
	ctx          context.Context
	credentials  credentials.Context
	reposcheme   RepositoryTypeScheme
	spechandlers RepositorySpecHandlers
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

func (b Builder) WithRepositoyTypeScheme(scheme RepositoryTypeScheme) Builder {
	b.reposcheme = scheme
	return b
}

func (b Builder) WithRepositorySpecHandlers(reg RepositorySpecHandlers) Builder {
	b.spechandlers = reg
	return b
}

func (b Builder) Bound() (Context, context.Context) {
	c := b.New()
	return c, context.WithValue(b.getContext(), key, c)
}

func (b Builder) New(m ...datacontext.BuilderMode) Context {
	mode := datacontext.Mode(m...)
	ctx := b.getContext()

	if b.credentials == nil {
		var ok bool
		b.credentials, ok = credentials.DefinedForContext(ctx)
		if !ok && mode != datacontext.MODE_SHARED {
			b.credentials = credentials.New(mode)
		} else {
			b.credentials = credentials.FromContext(ctx)
		}
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
	return datacontext.SetupContext(mode, newContext(b.credentials, b.reposcheme, b.spechandlers, b.credentials))
}
