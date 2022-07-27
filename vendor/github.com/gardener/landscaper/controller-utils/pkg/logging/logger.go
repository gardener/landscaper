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
	internal logr.Logger
}

// loggerContextKey is a key for finding a logger in the context
type loggerContextKey struct{}

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
	return l.internal.GetSink().Enabled(levelToVerbosity(lvl))
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

// FromContext tries to fetch a logger from the context.
// If that fails, it returns the result of logr.FromContext,
// with the logger being wrapped into our Logger struct.
func FromContext(ctx context.Context) (Logger, error) {
	if log, ok := ctx.Value(loggerContextKey{}).(Logger); ok {
		return log, nil
	}
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

func Discard() Logger {
	return Wrap(logr.Discard())
}

// ADDITIONAL FUNCTIONS

// Wrap constructs a new Logger, using the provided logr.Logger internally.
func Wrap(log logr.Logger) Logger {
	return Logger{internal: log}
}

// Debug logs a message at DEBUG level.
func (l Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.internal.V(levelToVerbosity(DEBUG)).Info(msg, keysAndValues...)
}

// Logr returns the internal logr.Logger.
func (l Logger) Logr() logr.Logger {
	return l.internal
}

// StartReconcile fetches the logger from the context and adds the reconciled resource.
// It also logs a 'start reconcile' message.
func StartReconcile(ctx context.Context, req reconcile.Request) (Logger, error) {
	log, err := FromContext(ctx)
	if err != nil {
		return Logger{}, fmt.Errorf("unable to get logger from context: %w", err)
	}
	log = log.WithValues(lc.KeyReconciledResource, req.NamespacedName)
	log.Info(lc.MsgStartReconcile)
	return log, nil
}
