// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package simplelogger

import (
	"fmt"
	"io"
	"time"

	"github.com/go-logr/logr"
)

// NewIOLogger creates a new logger that logs to the given io.Writer.
// The error is ignored.
// This logger is meant to be used with a testsuite.
func NewIOLogger(writer io.Writer) *IOLogger {
	return &IOLogger{
		name:   "",
		writer: writer,
	}
}

var _ logr.Logger = &IOLogger{}

// IOLogger is a Logger that writes all messages to the given io.Writer.
// This is a simple implementation so all levels are logged.
type IOLogger struct {
	name           string
	writer         io.Writer
	keysAndValues  []interface{}
	withTimestamps bool
}

func (l *IOLogger) WithTimestamps() *IOLogger {
	l.withTimestamps = true
	return l
}

func (l IOLogger) Enabled() bool {
	return true
}

func (l *IOLogger) Info(msg string, keysAndValues ...interface{}) {
	msg = fmt.Sprintf("%s - %#v\n", msg, append(l.keysAndValues, keysAndValues...))
	if len(l.name) != 0 {
		msg = l.name + "| " + msg
	}
	if l.withTimestamps {
		msg = time.Now().Format(time.RFC3339Nano) + " | " + msg
	}
	fmt.Fprint(l.writer, msg)
}

func (l *IOLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	msg = fmt.Sprintf("%s %s - %#v\n", err.Error(), msg, append(l.keysAndValues, keysAndValues...))
	if len(l.name) != 0 {
		msg = l.name + "| " + msg
	}
	if l.withTimestamps {
		msg = time.Now().Format(time.RFC3339Nano) + " | " + msg
	}
	fmt.Fprintf(l.writer, "error %s", msg)
}

func (l *IOLogger) V(level int) logr.Logger {
	return l
}

func (l *IOLogger) WithValues(keysAndValues ...interface{}) logr.Logger {
	return &IOLogger{
		name:           l.name,
		writer:         l.writer,
		keysAndValues:  append(l.keysAndValues, keysAndValues...),
		withTimestamps: l.withTimestamps,
	}
}

func (l IOLogger) WithName(name string) logr.Logger {
	return &IOLogger{
		name:           name,
		writer:         l.writer,
		keysAndValues:  l.keysAndValues,
		withTimestamps: l.withTimestamps,
	}
}

// Verify that it actually implements the interface
var _ logr.Logger = &IOLogger{}
