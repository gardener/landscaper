// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package simplelogger

import (
	"fmt"
	"io"

	"github.com/go-logr/logr"
)

// NewIOLogger creates a new logger that logs to the given io.Writer.
// The error is ignored.
// This logger is meant to be used with a testsuite.
func NewIOLogger(writer io.Writer) logr.Logger {
	return IOLogger{
		name:   "",
		writer: writer,
	}
}

// IOLogger is a Logger that writes all messages to the given io.Writer.
// This is a simple implementation so all levels are logged.
type IOLogger struct {
	name   string
	writer io.Writer
}

func (l IOLogger) Enabled() bool {
	return true
}

func (l IOLogger) Info(msg string, keysAndValues ...interface{}) {
	fmt.Fprintf(l.writer, "%s: %s - %#v", l.name, msg, keysAndValues)
}

func (l IOLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	fmt.Fprintf(l.writer, "error %s: %s %s - %#v", l.name, err.Error(), msg, keysAndValues)
}

func (l IOLogger) V(level int) logr.Logger {
	return l
}

func (l IOLogger) WithValues(keysAndValues ...interface{}) logr.Logger {
	return l
}

func (l IOLogger) WithName(name string) logr.Logger {
	return IOLogger{
		name:   name,
		writer: l.writer,
	}
}

// Verify that it actually implements the interface
var _ logr.Logger = IOLogger{}
