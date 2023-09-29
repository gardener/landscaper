// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package datacontext

import (
	"context"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext/action/api"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/action/handlers"
)

type Builder struct {
	ctx        context.Context
	attributes Attributes
	actions    handlers.Registry
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

func (b Builder) WithAttributes(paranetAttr Attributes) Builder {
	b.attributes = paranetAttr
	return b
}

func (b Builder) WithActionHandlers(hdlrs handlers.Registry) Builder {
	b.actions = hdlrs
	return b
}

func (b Builder) Bound() (Context, context.Context) {
	c := b.New()
	return c, context.WithValue(b.getContext(), key, c)
}

func (b Builder) New(m ...BuilderMode) Context {
	mode := Mode(m...)

	if b.actions == nil {
		switch mode {
		case MODE_INITIAL:
			b.actions = handlers.NewRegistry(api.NewActionTypeRegistry())
		case MODE_CONFIGURED:
			b.actions = handlers.NewRegistry(api.DefaultRegistry().Copy())
			handlers.DefaultRegistry().AddTo(b.actions)
		case MODE_EXTENDED:
			b.actions = handlers.NewRegistry(api.DefaultRegistry(), handlers.DefaultRegistry())
		case MODE_DEFAULTED:
			fallthrough
		case MODE_SHARED:
			b.actions = handlers.DefaultRegistry()
		}
	}

	return NewWithActions(nil, b.actions)
}
