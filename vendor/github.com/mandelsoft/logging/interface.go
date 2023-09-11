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

// FieldKeyRealm is the name of the logr field set to the realm of a logging
// message.
const FieldKeyRealm = "realm"

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

// ParseLevel maps a string representation of a log level to
// it internal value. It also accepts the representation as
// number.
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

// LevelName returns the logical name of a log level.
// It can be parsed again with ParseLevel.
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

// Logger is the main logging interface.
// It is used to issue log messages.
// Additionally, it provides methods
// to create loggers with extend names
// and key/value pairs.
type Logger interface {
	// LogError logs a given error with additional context.
	LogError(err error, msg string, keypairs ...interface{})
	// Error logs an error message.
	Error(msg string, keypairs ...interface{})
	// Warn logs a warning message.
	Warn(msg string, keypairs ...interface{})
	// Error logs an info message
	Info(msg string, keypairs ...interface{})
	// Debug logs a debug message.
	Debug(msg string, keypairs ...interface{})
	// Trace logs an trace message.
	Trace(msg string, keypairs ...interface{})

	// NewName return a new logger with an extended name,
	// but the same logging activation.
	WithName(name string) Logger
	// WithValues return a new logger with more standard key/value pairs,
	// but the same logging activation.
	WithValues(keypairs ...interface{}) Logger

	// Enabled check whether the logger is active for a dedicated level.
	Enabled(level int) bool

	// V returns a logr logger with the same activation state
	// like the actual logger at call time.
	V(delta int) logr.Logger
}

// UnboundLogger is a logger, which is never bound to
// the settings of a matching rule at the time of
// its creation. It therefore always reflects the
// state of the rule settings valid at the time
// of issuing log messages. They are more expensive than regular
// loggers, but they can be created and configured once
// and stored in long living variables.
//
// When passing loggers down a dynamic call tree, to control
// the logging here, only temporary (bound) loggers should be used
// (as provided by a logging context) to improve performance.
//
// Such a logger can be reused for multiple independent call trees
// without losing track to the config.
// Regular loggers provided by a context keep their setting from the
// matching rule valid during its creation.
//
// An unbound logger can be created with function DynamicLogger
// for a logging context.
type UnboundLogger interface {
	Logger
	WithContext(messageContext ...MessageContext) UnboundLogger
	BoundLogger() Logger
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

// UpdatableRule is the optional interface for a rule
// which might replace an old one.
// If a rule decides to supersede an old one when
// adding to a ruleset is may return true.
// Typically, a rule should only supersede rules
// of its own type.
type UpdatableRule interface {
	Rule
	MatchRule(Rule) bool
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

	// AddRule adds a rule to the actual context.
	// It may decide to supersede already existing rules. In
	// such case all matching rules (interface UpdatableRule)
	// will be deleted from the active rule set.
	AddRule(...Rule)
	// ResetRules deletes the actual rule set.
	ResetRules()
	// AddRulesTo add the actual rules to another logging context.
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

// LoggingContext returns a default logging context for an
// arbitrary object. If the object supports the LoggingProvider
// interface, it is used to determine the context. If not,
// the default logging context is returned.
func LoggingContext(p interface{}) Context {
	if p != nil {
		if cp, ok := p.(ContextProvider); ok {
			return cp.LoggingContext()
		}
	}
	return DefaultContext()
}

// Attacher is an optional interface, which can be implemented by a dedicated
// type of message context. If available it is used to enrich the attributes
// of a determined logger to describe the given context.
type Attacher interface {
	Attach(l Logger) Logger
}

// Realm is some kind of tag, which can be used as
// message context or logging condition.
// If used as message context it will be attached to
// the logging message as additional logger name.
type Realm = realm

var (
	_ Condition = Realm("")
	_ Attacher  = Realm("")
)

// RealmPrefix is used as logging condition to
// match a realm of the message context by
// checking its value to be a path prefix.
type RealmPrefix = realmprefix

var _ Condition = RealmPrefix("")

// Attribute is a key/value pair usable
// as logging condition or message context.
// If used as message context it will be attached to
// the logging message as additional value.
type Attribute interface {
	Condition
	Attacher

	Name() string
	Value() interface{}
}

// Tag is a simple string value, which can be used as
// message context or logging condition.
// If used as message context it will not be attached to
// the logging message at all.
type Tag = tag

var _ Condition = Tag("")

// Name is a simple string value, which can be used as
// message context.
// It will not be attached to the logger's name.
type Name = name

var _ Attacher = Name("")
