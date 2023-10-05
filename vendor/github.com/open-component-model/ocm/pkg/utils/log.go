// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"

	"github.com/go-logr/logr"
)

// logContextKey is the unique key for storing additional logging contexts.
type logContextKey struct{}

// LogContextValues describes the context values.
type LogContextValues map[string]interface{}

func ContextWithLogContextValues(parent context.Context) (context.Context, *LogContextValues) {
	vals := &LogContextValues{}
	return context.WithValue(parent, logContextKey{}, vals), vals
}

// LogContextValuesFromContext returns the context values of a ctx.
// If nothing is defined nil is returned.
func LogContextValuesFromContext(ctx context.Context) *LogContextValues {
	c, ok := ctx.Value(logContextKey{}).(*LogContextValues)
	if !ok {
		return nil
	}
	return c
}

// AddLogContextValue adds a key value pair to the logging context.
// If none is defined it will be added.
func AddLogContextValue(ctx context.Context, key string, value interface{}) context.Context {
	logCtx := LogContextValuesFromContext(ctx)
	if logCtx == nil {
		ctx, logCtx = ContextWithLogContextValues(ctx)
	}
	(*logCtx)[key] = value
	return ctx
}

// ctxSink defines a log sink that injects the provided context values
// and delegates the actual logging to a delegate.
type ctxSink struct {
	logr.LogSink
	ctx *LogContextValues
}

// LoggerWithContextContextValues creates a new context logger that delegates the actual requests to the delegate
// but injects the context log values.
func LoggerWithContextContextValues(ctx context.Context, delegate logr.Logger) logr.Logger {
	val := LogContextValuesFromContext(ctx)
	if val == nil {
		return delegate
	}
	return delegate.WithSink(newWithContextValues(val, delegate.GetSink()))
}

func newWithContextValues(ctx *LogContextValues, del logr.LogSink) logr.LogSink {
	return &ctxSink{
		LogSink: del,
		ctx:     ctx,
	}
}

func (c ctxSink) Error(err error, msg string, keysAndValues ...interface{}) {
	// append log context values
	if c.ctx == nil {
		c.LogSink.Error(err, msg, keysAndValues...)
		return
	}
	for key, val := range *c.ctx {
		keysAndValues = append(keysAndValues, key, val)
	}
	c.LogSink.Error(err, msg, keysAndValues...)
}
