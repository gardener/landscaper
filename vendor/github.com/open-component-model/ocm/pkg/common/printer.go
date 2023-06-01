// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/mandelsoft/logging"
)

type Flusher interface {
	Flush() error
}

func Flush(o interface{}) error {
	if f, ok := o.(Flusher); ok {
		return f.Flush()
	}
	return nil
}

type Printer interface {
	io.Writer
	Printf(msg string, args ...interface{}) (int, error)

	AddGap(gap string) Printer
}

type FlushingPrinter interface {
	Printer
	Flusher
}

type printerState struct {
	pending bool
}

type printer struct {
	writer io.Writer
	gap    string
	state  *printerState
}

func NewPrinter(writer io.Writer) Printer {
	return &printer{writer: writer, state: &printerState{true}}
}

func NewBufferedPrinter() (Printer, *bytes.Buffer) {
	buf := bytes.NewBuffer(nil)
	return NewPrinter(buf), buf
}

func (p *printer) AddGap(gap string) Printer {
	return &printer{
		writer: p.writer,
		gap:    p.gap + gap,
		state:  p.state,
	}
}

func (p *printer) Write(data []byte) (int, error) {
	if p.writer == nil {
		return 0, nil
	}
	s := strings.ReplaceAll(string(data), "\n", "\n"+p.gap)
	if strings.HasSuffix(s, "\n"+p.gap) {
		p.state.pending = true
		s = s[:len(s)-len(p.gap)]
	}
	return p.writer.Write([]byte(s))
}

func (p *printer) printf(msg string, args ...interface{}) (int, error) {
	if p == nil || p.writer == nil {
		return 0, nil
	}
	if p.gap == "" {
		return fmt.Fprintf(p.writer, msg, args...)
	}
	if p.state.pending {
		msg = p.gap + msg
	}
	data := fmt.Sprintf(msg, args...)
	p.state.pending = false
	return p.Write([]byte(data))
}

func (p *printer) Printf(msg string, args ...interface{}) (int, error) {
	return p.printf(msg, args...)
}

////////////////////////////////////////////////////////////////////////////////

type loggingPrinter struct {
	log     logging.Logger
	gap     string
	pending string
}

// NewLoggingPrinter returns a printer logging the output to an
// info-level log.
// It should not be used to print binary data, but text data, only.
func NewLoggingPrinter(log logging.Logger) FlushingPrinter {
	return &loggingPrinter{log: log}
}

func (p *loggingPrinter) AddGap(gap string) Printer {
	return &loggingPrinter{
		log: p.log,
		gap: p.gap + gap,
	}
}

func (p *loggingPrinter) Write(data []byte) (int, error) {
	if p.log == nil {
		return 0, nil
	}
	s := strings.Split(p.pending+string(data), "\n")
	if !strings.HasSuffix(string(data), "\n") {
		p.pending = s[len(s)-1]
	} else {
		p.pending = ""
	}
	s = s[:len(s)-1]
	for _, l := range s {
		p.log.Info(l)
	}
	return len(data), nil
}

func (p *loggingPrinter) Printf(msg string, args ...interface{}) (int, error) {
	if p.log == nil {
		return 0, nil
	}
	return p.Write([]byte(fmt.Sprintf(msg, args...)))
}

func (p *loggingPrinter) Flush() error {
	if p.pending != "" {
		p.log.Info(p.pending)
		p.pending = ""
	}
	return nil
}
