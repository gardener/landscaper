/*
 * Copyright 2022 Mandelsoft. All rights reserved.
 *  This file is licensed under the Apache Software License, v. 2 except as noted
 *  otherwise in the LICENSE file
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package logging

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
)

// These are the different logging levels. You can set the logging level to log
// on your instance of logger.
const (
	// None level. No logging,
	None = iota
	// ErrorLevel level. Logs. Used for errors that should definitely be noted.
	// Commonly used for hooks to send errors to an error tracking service.
	ErrorLevel
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel
	// InfoLevel level. General operational entries about what's going on inside the
	// application.
	InfoLevel
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	DebugLevel
	// TraceLevel level. Designates finer-grained informational events than the Debug.
	TraceLevel
)

func ParseLevel(s string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "error":
		return ErrorLevel, nil
	case "warn":
		return WarnLevel, nil
	case "info":
		return InfoLevel, nil
	case "debug":
		return DebugLevel, nil
	case "trace":
		return TraceLevel, nil
	default:
		v, err := strconv.ParseInt(s, 10, 32)
		if err != nil || v < 0 {
			return 0, fmt.Errorf("invalid log level %q", s)
		}
		return int(v), nil
	}
}

func LevelName(l int) string {
	switch l {
	case ErrorLevel:
		return "Error"
	case WarnLevel:
		return "Warn"
	case InfoLevel:
		return "Info"
	case DebugLevel:
		return "Debug"
	case TraceLevel:
		return "Trace"
	default:
		return fmt.Sprintf("%d", l)
	}
}

type Logger interface {
	LogError(err error, msg string, keypairs ...interface{})
	Error(msg string, keypairs ...interface{})
	Warn(msg string, keypairs ...interface{})
	Info(msg string, keypairs ...interface{})
	Debug(msg string, keypairs ...interface{})
	Trace(msg string, keypairs ...interface{})

	WithName(name string) Logger
	WithValues(keypairs ...interface{}) Logger
	Enabled(level int) bool

	V(delta int) logr.Logger
}

// MessageContext is an object providing context information for
// a log message.
type MessageContext interface {
}

// Condition matches a given message context.
// It returns true, if the condition matches the context.
type Condition interface {
	Match(...MessageContext) bool
}

// Rule matches a given message context and returns
// an appropriate logger
type Rule interface {
	Match(SinkFunc, ...MessageContext) Logger
}

// ContextProvider is able to provide access to a logging context.
type ContextProvider interface {
	LoggingContext() Context
}

type LevelFunc func() int
type SinkFunc func() logr.LogSink

// Context describes the interface of a logging context.
// A logging context determines effective loggers for
// a given message context based on a set of rules used
// to map a message context to an effective logger.
type Context interface {
	ContextProvider

	// GetSink returns the effective logr.LOgSink used as base logger
	// for this context.
	// In case of a nested context, this is the locally set sink, if set,
	// or the sink of the base context.
	GetSink() logr.LogSink
	// GetDefaultLevel returns the default log level effective, if no rule matches.
	// These may be locally defined rules, or, in case of a nested logger,
	// rules of the base context, also.
	GetDefaultLevel() int
	// GetDefaultLogger return the effective default logger used if no rule matches
	// a message context.
	GetDefaultLogger() Logger
	// SetDefaultLevel sets the default logging level to use for provided
	// Loggers, if no rule matches.
	SetDefaultLevel(level int)
	// SetBaseLogger sets a new base logger.
	// If the optional parameter plain is set to true, the base logger
	// is rebased to the absolute logr level 0.
	// Otherwise, the base log level is taken from the given logger. This means
	// ErrorLevel is mapped to the log level of the given logger.
	// Although the error output is filtered by this log level by the
	// original sink, error level output, if enabled, is passed as Error to the sink.
	SetBaseLogger(logger logr.Logger, plain ...bool)

	AddRule(...Rule)
	ResetRules()
	AddRulesTo(ctx Context)

	// WithContext provides a new logging Context enriched by the given standard
	// message context
	WithContext(messageContext ...MessageContext) Context
	// Logger return the effective logger for the given message context.
	Logger(...MessageContext) Logger
	// V returns the effective logr.Logger for the given message context with
	// the given base level.
	V(level int, mctx ...MessageContext) logr.Logger

	// Evaluate returns the effective logger for the given message context
	// based on the given logr.LogSink.
	Evaluate(SinkFunc, ...MessageContext) Logger

	// Tree provides an interface for the context intended for
	// context implementations to work together in a context tree.
	Tree() ContextSupport
}

// Attacher is an optional interface, which can be implemented by a dedicated
// type of message context. If available it is used to enrich the attributes
// of a determined logger to describe the given context.
type Attacher interface {
	Attach(l Logger) Logger
}

type Realm interface {
	Condition
	Attacher

	Name() string
}

type RealmPrefix interface {
	Condition
	Attacher

	Name() string
	IsPrefix() bool
}

type Attribute interface {
	Condition
	Attacher

	Name() string
	Value() interface{}
}

type Tag interface {
	Condition

	Name() string
}
