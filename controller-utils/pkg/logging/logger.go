// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
)

type Logger struct {
	internal    logr.Logger
	initialized bool
}

type LogLevel int

const (
	unknown_level LogLevel = iota // dummy value to detect if not set
	ERROR
	INFO
	DEBUG
)

func (l LogLevel) String() string {
	switch l {
	case ERROR:
		return "ERROR"
	case INFO:
		return "INFO"
	case DEBUG:
		return "DEBUG"
	}
	return "UNKNOWN"
}

func ParseLogLevel(raw string) (LogLevel, error) {
	upper := strings.ToUpper(raw)
	switch upper {
	case "ERROR":
		return ERROR, nil
	case "INFO":
		return INFO, nil
	case "DEBUG":
		return DEBUG, nil
	}
	return INFO, fmt.Errorf("unknown log level '%s', valid values are: [%s] (case-insensitive)", raw, strings.Join([]string{ERROR.String(), INFO.String(), DEBUG.String()}, ", "))
}

type LogFormat int

const (
	unknown_format LogFormat = iota
	TEXT
	JSON
)

func (f LogFormat) String() string {
	switch f {
	case TEXT:
		return "TEXT"
	case JSON:
		return "JSON"
	}
	return "UNKNOWN"
}

func ParseLogFormat(raw string) (LogFormat, error) {
	upper := strings.ToUpper(raw)
	switch upper {
	case "TEXT":
		return TEXT, nil
	case "JSON":
		return JSON, nil
	}
	return TEXT, fmt.Errorf("unknown log format '%s', valid values are: [%s] (case-insensitive)", raw, strings.Join([]string{TEXT.String(), JSON.String()}, ", "))
}

// LOGR WRAPPER FUNCTIONS

// Enabled tests whether logging at the provided level is enabled.
// This deviates from the logr Enabled() function, which doesn't take an argument.
func (l Logger) Enabled(lvl LogLevel) bool {
	return l.internal.GetSink() != nil && l.internal.GetSink().Enabled(levelToVerbosity(lvl))
}

// Info logs a non-error message with the given key/value pairs as context.
//
// The msg argument should be used to add some constant description to
// the log line.  The key/value pairs can then be used to add additional
// variable information.  The key/value pairs should alternate string
// keys and arbitrary values.
func (l Logger) Info(msg string, keysAndValues ...interface{}) {
	l.internal.V(levelToVerbosity(INFO)).Info(msg, keysAndValues...)
}

// Error logs an error, with the given message and key/value pairs as context.
// It functions similarly to calling Info with the "error" named value, but may
// have unique behavior, and should be preferred for logging errors (see the
// package documentations for more information).
//
// The msg field should be used to add context to any underlying error,
// while the err field should be used to attach the actual error that
// triggered this log line, if present.
func (l Logger) Error(err error, msg string, keysAndValues ...interface{}) {
	l.internal.Error(err, msg, keysAndValues...)
}

// WithValues adds some key-value pairs of context to a logger.
// See Info for documentation on how key/value pairs work.
func (l Logger) WithValues(keysAndValues ...interface{}) Logger {
	return Wrap(l.internal.WithValues(keysAndValues...))
}

// WithName adds a new element to the logger's name.
// Successive calls with WithName continue to append
// suffixes to the logger's name.  It's strongly recommended
// that name segments contain only letters, digits, and hyphens
// (see the package documentation for more information).
func (l Logger) WithName(name string) Logger {
	return Wrap(l.internal.WithName(name))
}

// FromContext wraps the result of logr.FromContext into a logging.Logger.
func FromContext(ctx context.Context) (Logger, error) {
	log, err := logr.FromContext(ctx)
	return Wrap(log), err
}

// FromContextOrDiscard works like FromContext, but it will return a discard logger if no logger is found in the context.
func FromContextOrDiscard(ctx context.Context) Logger {
	log, err := FromContext(ctx)
	if err != nil {
		return Discard()
	}
	return log
}

// NewContext is a wrapper for logr.NewContext.
func NewContext(ctx context.Context, log Logger) context.Context {
	return logr.NewContext(ctx, log.Logr())
}

// Discard is a wrapper for logr.Discard.
func Discard() Logger {
	return Wrap(logr.Discard())
}

// ADDITIONAL FUNCTIONS

// Wrap constructs a new Logger, using the provided logr.Logger internally.
func Wrap(log logr.Logger) Logger {
	return Logger{internal: log, initialized: true}
}

// FromContextOrNew tries to fetch a logger from the context.
// It is expected that a logger is contained in the context. If retrieving it fails, a new logger will be created and an error is logged.
// keysAndValuesFallback contains keys and values which will only be added if the logger could not be retrieved and a new one had to be created.
// The key-value-pairs from keysAndValues will always be added.
// A new context, containing the created logger, will be returned.
// The function panics if the logger cannot be fetched from the context and creating a new one fails.
func FromContextOrNew(ctx context.Context, keysAndValuesFallback []interface{}, keysAndValues ...interface{}) (Logger, context.Context) {
	log, err := FromContext(ctx)
	if err != nil {
		newLogger, err2 := GetLogger()
		if err2 != nil {
			panic(err2)
		}

		newLogger = newLogger.WithValues(keysAndValuesFallback...).WithValues(keysAndValues...)
		newLogger.Error(err2, "unable to fetch logger from context")
		ctx = NewContext(ctx, newLogger)
		return newLogger, ctx
	}
	if len(keysAndValues) > 0 {
		log = log.WithValues(keysAndValues...)
		ctx = NewContext(ctx, log)
	}
	return log, ctx
}

// FromContextWithFallback tries to fetch a logger from the context.
// If that fails, the provided fallback logger is used instead.
// It returns the fetched logger, enriched with the given key-value-pairs, and a context containing this new logger.
func FromContextWithFallback(ctx context.Context, fallback Logger, keysAndValues ...interface{}) (Logger, context.Context) {
	log, err := FromContext(ctx)
	if err != nil {
		log = fallback
	}
	log = log.WithValues(keysAndValues...)
	ctx = NewContext(ctx, log)
	return log, ctx
}

// NewContextWithDiscard adds a discard logger to the given context and returns the new context.
func NewContextWithDiscard(ctx context.Context) context.Context {
	return NewContext(ctx, Discard())
}

// Debug logs a message at DEBUG level.
func (l Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.internal.V(levelToVerbosity(DEBUG)).Info(msg, keysAndValues...)
}

// Log logs at the given log level. It can be used to log at dynamically determined levels.
func (l Logger) Log(lvl LogLevel, msg string, keysAndValues ...interface{}) {
	switch lvl {
	case ERROR:
		l.Error(nil, msg, keysAndValues...)
	case DEBUG:
		l.Debug(msg, keysAndValues...)
	default:
		l.Info(msg, keysAndValues...)
	}
}

// IsInitialized returns true if the logger is ready to be used and
// false if it is an 'empty' logger (e.g. created by Logger{}).
func (l Logger) IsInitialized() bool {
	return l.initialized
}

// Reconciles is meant to be used for the logger initialization for controllers.
// It is a wrapper for WithName(name).WithValues(lc.KeyReconciledResourceKind, reconciledResource).
func (l Logger) Reconciles(name, reconciledResource string) Logger {
	log := l
	if len(name) != 0 {
		log = log.WithName(name)
	}
	if len(reconciledResource) != 0 {
		log = log.WithValues(lc.KeyReconciledResourceKind, reconciledResource)
	}
	return log
}

// Logr returns the internal logr.Logger.
func (l Logger) Logr() logr.Logger {
	return l.internal
}

// StartReconcileFromContext fetches the logger from the context and adds the reconciled resource.
// It also logs a 'start reconcile' message.
// The returned context contains the enriched logger.
func StartReconcileFromContext(ctx context.Context, req reconcile.Request) (Logger, context.Context, error) {
	log, err := FromContext(ctx)
	if err != nil {
		return Logger{}, ctx, fmt.Errorf("unable to get logger from context: %w", err)
	}
	log, ctx = log.StartReconcileAndAddToContext(ctx, req)
	return log, ctx, nil
}

// StartReconcile works like StartReconcileFromContext, but it is called on an existing logger instead of fetching one from the context.
func (l Logger) StartReconcile(req reconcile.Request, keysAndValues ...interface{}) Logger {
	newLogger := l.WithValues(lc.KeyReconciledResource, req.NamespacedName).WithValues(keysAndValues...)
	newLogger.Debug(lc.MsgStartReconcile)
	return newLogger
}

// StartReconcileAndAddToContext works like StartReconcile, but additionally returns a context with the new logger.
func (l Logger) StartReconcileAndAddToContext(ctx context.Context, req reconcile.Request, keysAndValues ...interface{}) (Logger, context.Context) {
	newLogger := l.StartReconcile(req, keysAndValues...)
	ctx = NewContext(ctx, newLogger)
	return newLogger, ctx
}

// MustStartReconcileFromContext is a helper function for easily getting a logger at the beginning of a reconciliation.
// It is basically a combination of FromContextOrNew, which tries to fetch the logger from the context, or creates a new one, if that fails,
// and StartReconcileAndAddToContext, which logs the beginning of the reconciliation and returns a new context containing the configured logger.
func MustStartReconcileFromContext(ctx context.Context, req reconcile.Request, keysAndValuesFallback []interface{}, keysAndValues ...interface{}) (Logger, context.Context) {
	log, ctx := FromContextOrNew(ctx, keysAndValuesFallback, keysAndValues...)
	return log.StartReconcileAndAddToContext(ctx, req)
}

// WithValuesAndContext works like WithValues, but also adds the logger directly to a context and returns the new context.
func (l Logger) WithValuesAndContext(ctx context.Context, keysAndValues ...interface{}) (Logger, context.Context) {
	log := l.WithValues(keysAndValues...)
	ctx = NewContext(ctx, log)
	return log, ctx
}

// WithNameAndContext works like WithName, but also adds the logger directly to a context and returns the new context.
func (l Logger) WithNameAndContext(ctx context.Context, name string) (Logger, context.Context) {
	log := l.WithName(name)
	ctx = NewContext(ctx, log)
	return log, ctx
}
