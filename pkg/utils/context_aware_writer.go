// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"io"
)

// ContextAwareWriter wraps a context and an io writer.
// When the context is cancelled, the write function will return early before calling write on the io writer.
type ContextAwareWriter struct {
	// ctx is the wrapping context
	ctx context.Context
	// writer is the underlying io writer
	writer io.Writer
}

// Write will check the wrapping context for errors.
// If the context doesn't return any error, the underlying io writer is called.
func (w *ContextAwareWriter) Write(p []byte) (n int, err error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	return w.writer.Write(p)
}

// NewContextAwareWriter creates a new context aware writer.
func NewContextAwareWriter(ctx context.Context, writer io.Writer) io.Writer {
	return &ContextAwareWriter{
		ctx:    ctx,
		writer: writer,
	}
}
