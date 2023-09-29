// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package compression

import (
	"io"

	"github.com/klauspost/pgzip"
)

// GzipAlgorithmName is the name used by pkg/compression.Gzip.
// NOTE: Importing only this /types package does not inherently guarantee a Gzip algorithm
// will actually be available. (In fact it is intended for this types package not to depend
// on any of the implementations.)
const GzipAlgorithmName = "gzip"

var Gzip = NewAlgorithm(GzipAlgorithmName, GzipAlgorithmName,
	[]byte{0x1F, 0x8B, 0x08}, gzipDecompressor, gzipCompressor)

func init() {
	Register(Gzip)
}

// gzipDecompressor is a DecompressorFunc for the gzip compression algorithm.
func gzipDecompressor(r io.Reader) (io.ReadCloser, error) {
	return pgzip.NewReader(r)
}

// gzipCompressor is a CompressorFunc for the gzip compression algorithm.
func gzipCompressor(r io.Writer, metadata map[string]string, level *int) (io.WriteCloser, error) {
	if level != nil {
		return pgzip.NewWriterLevel(r, *level)
	}
	return pgzip.NewWriter(r), nil
}
