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
func NewIOLogger(writer io.Writer) logr.Logger {
	return logr.New(newIOLogSink(writer))
}

var _ logr.LogSink = &ioLogSink{}

func WithTimestamps(log logr.Logger) logr.Logger {
	if ls, ok := log.GetSink().(*ioLogSink); ok {
		log = log.WithSink(newIOLogSink(ls.writer).WithTimestamps())
	}
	return log
}

// ioLogSink is a Logger implementation that writes all messages to the given io.Writer.
// This is a simple implementation so all levels are logged.
type ioLogSink struct {
	name           string
	writer         io.Writer
	keysAndValues  []interface{}
	withTimestamps bool
}

func newIOLogSink(writer io.Writer) *ioLogSink {
	return &ioLogSink{
		name:   "",
		writer: writer,
	}
}

func (l *ioLogSink) WithTimestamps() logr.LogSink {
	l.withTimestamps = true
	return l
}

func (l *ioLogSink) Init(info logr.RuntimeInfo) {}

func (l ioLogSink) Enabled(level int) bool {
	return true
}

func (l *ioLogSink) Info(level int, msg string, keysAndValues ...interface{}) {
	msg = fmt.Sprintf("%s - %#v\n", msg, append(l.keysAndValues, keysAndValues...))
	if len(l.name) != 0 {
		msg = l.name + "| " + msg
	}
	if l.withTimestamps {
		msg = time.Now().Format(time.RFC3339Nano) + " | " + msg
	}
	fmt.Fprint(l.writer, msg)
}

func (l *ioLogSink) Error(err error, msg string, keysAndValues ...interface{}) {
	msg = fmt.Sprintf("%s %s - %#v\n", err.Error(), msg, append(l.keysAndValues, keysAndValues...))
	if len(l.name) != 0 {
		msg = l.name + "| " + msg
	}
	if l.withTimestamps {
		msg = time.Now().Format(time.RFC3339Nano) + " | " + msg
	}
	fmt.Fprintf(l.writer, "error %s", msg)
}

func (l *ioLogSink) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return &ioLogSink{
		name:           l.name,
		writer:         l.writer,
		keysAndValues:  append(l.keysAndValues, keysAndValues...),
		withTimestamps: l.withTimestamps,
	}
}

func (l ioLogSink) WithName(name string) logr.LogSink {
	return &ioLogSink{
		name:           name,
		writer:         l.writer,
		keysAndValues:  l.keysAndValues,
		withTimestamps: l.withTimestamps,
	}
}
