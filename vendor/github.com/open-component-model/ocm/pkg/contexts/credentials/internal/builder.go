// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"

	"github.com/open-component-model/ocm/pkg/contexts/config"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
)

type Builder struct {
	ctx        context.Context
	config     config.Context
	reposcheme RepositoryTypeScheme
	matchers   IdentityMatcherRegistry
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

func (b Builder) WithConfig(ctx config.Context) Builder {
	b.config = ctx
	return b
}

func (b Builder) WithRepositoyTypeScheme(scheme RepositoryTypeScheme) Builder {
	b.reposcheme = scheme
	return b
}

func (b Builder) WithStandardConumerMatchers(matchers IdentityMatcherRegistry) Builder {
	b.matchers = matchers
	return b
}

func (b Builder) Bound() (Context, context.Context) {
	c := b.New()
	return c, context.WithValue(b.getContext(), key, c)
}

func (b Builder) New(m ...datacontext.BuilderMode) Context {
	mode := datacontext.Mode(m...)
	ctx := b.getContext()

	if b.config == nil {
		var ok bool
		b.config, ok = config.DefinedForContext(ctx)
		if !ok && mode != datacontext.MODE_SHARED {
			b.config = config.New(mode)
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
	if b.matchers == nil {
		b.matchers = StandardIdentityMatchers
	}
	return newContext(b.config, b.reposcheme, b.matchers, b.config.LoggingContext())
}
