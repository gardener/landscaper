// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package logcontext

import (
	"context"

	"github.com/go-logr/logr"
)

// logContextKey is the unique key for storing additional logging contexts
type logContextKey struct{}

// ContextValues describes the context values.
type ContextValues map[string]interface{}

func NewContext(parent context.Context) (context.Context, *ContextValues) {
	vals := &ContextValues{}
	return context.WithValue(parent, logContextKey{}, vals), vals
}

// FromContext returns the context values of a ctx.
// If nothing is defined nil is returned.
func FromContext(ctx context.Context) *ContextValues {
	c, ok := ctx.Value(logContextKey{}).(*ContextValues)
	if !ok {
		return nil
	}
	return c
}

// AddContextValue adds a key value pair to the logging context.
// If none is defined it will be added.
func AddContextValue(ctx context.Context, key string, value interface{}) context.Context {
	logCtx := FromContext(ctx)
	if logCtx == nil {
		ctx, logCtx = NewContext(ctx)
	}
	(*logCtx)[key] = value
	return ctx
}

// ctxLogger defines a logger that injects the provided context values
// and delegates the actual logging to a delegate.
type ctxLogger struct {
	l   logr.Logger
	ctx *ContextValues
}

// New creates a new context logger that delegates the actual requests to the delegate
// but injects the context log values.
func New(ctx context.Context, delegate logr.Logger) logr.Logger {
	val := FromContext(ctx)
	return newWithContextValues(val, delegate)
}

func newWithContextValues(ctx *ContextValues, del logr.Logger) *ctxLogger {
	return &ctxLogger{
		l:   del,
		ctx: ctx,
	}
}

func (c ctxLogger) Enabled() bool {
	return c.l.Enabled()
}

func (c ctxLogger) Info(msg string, keysAndValues ...interface{}) {
	c.l.Info(msg, keysAndValues...)
}

func (c ctxLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	// append log context values
	if c.ctx == nil {
		c.l.Info(msg, keysAndValues...)
		return
	}
	for key, val := range *c.ctx {
		keysAndValues = append(keysAndValues, key, val)
	}
	c.l.Error(err, msg, keysAndValues...)
}

func (c ctxLogger) V(level int) logr.Logger {
	return newWithContextValues(c.ctx, c.l.V(level))
}

func (c ctxLogger) WithValues(keysAndValues ...interface{}) logr.Logger {
	return newWithContextValues(c.ctx, c.l.WithValues(keysAndValues...))
}

func (c ctxLogger) WithName(name string) logr.Logger {
	return newWithContextValues(c.ctx, c.l.WithName(name))
}
