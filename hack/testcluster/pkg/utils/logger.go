// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"io"
	"os"
	"time"
)

// Logger provides a simplified, plain logging interface in contrast to
// logr.Logger
type Logger interface {
	Log(message string)
	Logln(message string)
	Logf(format string, a ...interface{})
	Logfln(format string, a ...interface{})
	WithTimestamp() Logger
}

// NewLoggerFromWriter creates a new logger that writes to the given writer
func NewLoggerFromWriter(writer io.Writer) Logger {
	tmpWriter := writer
	if tmpWriter == nil {
		tmpWriter = os.Stdout
	}
	return &logger{writer: tmpWriter}
}

// NewLogger creates a new logger that writes to stdout
func NewLogger() Logger {
	return &logger{
		writer: os.Stdout,
	}
}

func NewDiscardLogger() Logger {
	return &logger{
		writer: nil,
	}
}

type logger struct {
	enableTimestamp bool
	writer          io.Writer
}

func (l logger) Log(msg string) {
	if l.writer == nil {
		return
	}

	if !l.enableTimestamp {
		_, _ = fmt.Fprint(l.writer, msg)
		return
	}
	_, _ = fmt.Fprintf(l.writer, "%s: %s", time.Now().Format(time.RFC3339), msg)
}

func (l logger) Logln(format string) {
	l.Log(format + "\n")
}

func (l logger) Logf(format string, a ...interface{}) {
	l.Log(fmt.Sprintf(format, a...))
}

func (l logger) Logfln(format string, a ...interface{}) {
	l.Logln(fmt.Sprintf(format, a...))
}

func (l logger) WithTimestamp() Logger {
	return &logger{
		writer:          l.writer,
		enableTimestamp: true,
	}
}
