// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
)

type Builder struct {
	ctx        context.Context
	shared     datacontext.AttributesContext
	reposcheme ConfigTypeScheme
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

func (b Builder) WithSharedAttributes(ctx datacontext.AttributesContext) Builder {
	b.shared = ctx
	return b
}

func (b Builder) WithConfigTypeScheme(scheme ConfigTypeScheme) Builder {
	b.reposcheme = scheme
	return b
}

func (b Builder) Bound() (Context, context.Context) {
	c := b.New()
	return c, context.WithValue(b.getContext(), key, c)
}

func (b Builder) New(m ...datacontext.BuilderMode) Context {
	mode := datacontext.Mode(m...)
	ctx := b.getContext()

	if b.shared == nil {
		if mode == datacontext.MODE_SHARED {
			b.shared = datacontext.ForContext(ctx)
		} else {
			b.shared = datacontext.New(nil)
		}
	}
	if b.reposcheme == nil {
		switch mode {
		case datacontext.MODE_INITIAL:
			b.reposcheme = NewConfigTypeScheme(nil)
		case datacontext.MODE_CONFIGURED:
			b.reposcheme = NewConfigTypeScheme(nil)
			b.reposcheme.AddKnownTypes(DefaultConfigTypeScheme)
		case datacontext.MODE_EXTENDED:
			b.reposcheme = NewConfigTypeScheme(nil, DefaultConfigTypeScheme)
		case datacontext.MODE_DEFAULTED:
			fallthrough
		case datacontext.MODE_SHARED:
			b.reposcheme = DefaultConfigTypeScheme
		}
	}
	return datacontext.SetupContext(mode, newContext(b.shared, b.reposcheme, b.shared))
}
