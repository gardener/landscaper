// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package compression

import (
	"bytes"
	"errors"
	"io"
)

// algorithm is a default implementation for Algorithm that can be used for CompressStream
// based on Compression and Decompression functions.
type algorithm struct {
	name         string
	mime         string
	prefix       []byte // Initial bytes of a stream compressed using this algorithm, or empty to disable detection.
	decompressor DecompressorFunc
	compressor   CompressorFunc
}

// NewAlgorithm creates an Algorithm instance.
// This function exists so that Algorithm instances can only be created by code that
// is allowed to import this internal subpackage.
func NewAlgorithm(name, mime string, prefix []byte, decompressor DecompressorFunc, compressor CompressorFunc) Algorithm {
	return &algorithm{
		name:         name,
		mime:         mime,
		prefix:       prefix,
		decompressor: decompressor,
		compressor:   compressor,
	}
}

// Name returns the name for the compression algorithm.
func (c *algorithm) Name() string {
	return c.name
}

// InternalUnstableUndocumentedMIMEQuestionMark ???
// DO NOT USE THIS anywhere outside c/image until it is properly documented.
func (c *algorithm) InternalUnstableUndocumentedMIMEQuestionMark() string {
	return c.mime
}

// Compressor returns a compressor for the given stream according to this algorithm .
func (c *algorithm) Compressor(w io.Writer, meta map[string]string, level *int) (io.WriteCloser, error) {
	if meta == nil {
		meta = map[string]string{}
	}

	return c.compressor(w, meta, level)
}

// Decompressor returns a decompressor for the given stream according to this algorithm .
func (c *algorithm) Decompressor(r io.Reader) (io.ReadCloser, error) {
	return c.decompressor(r)
}

func (c *algorithm) Match(r MatchReader) (bool, error) {
	if len(c.prefix) == 0 {
		return false, nil
	}
	buf := make([]byte, len(c.prefix))
	n, err := io.ReadAtLeast(r, buf, len(buf))
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			err = nil
		}
		return false, err
	}
	return bytes.HasPrefix(buf[:n], c.prefix), nil
}
