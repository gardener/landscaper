// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

// This package has been initially taken from github.com/containers/image
// and modified to be provide a useful simple API based on
// an Algorithm interface

package compression

import (
	"io"
)

// CompressorFunc writes the compressed stream to the given writer using the specified compression level.
// The caller must call Close() on the stream (even if the input stream does not need closing!).
type CompressorFunc func(io.Writer, map[string]string, *int) (io.WriteCloser, error)

// DecompressorFunc returns the decompressed stream, given a compressed stream.
// The caller must call Close() on the decompressed stream (even if the compressed input stream does not need closing!).
type DecompressorFunc func(io.Reader) (io.ReadCloser, error)

// Algorithm is a compression algorithm provided and supported by pkg/compression.
// It can’t be supplied from the outside.
type Algorithm interface {
	Name() string
	Compressor(io.Writer, map[string]string, *int) (io.WriteCloser, error)
	Decompressor(io.Reader) (io.ReadCloser, error)
	Match(MatchReader) (bool, error)
}
