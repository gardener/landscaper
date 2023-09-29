// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package compression

import (
	"compress/bzip2"
	"fmt"
	"io"
)

// Bzip2AlgorithmName is the name used by pkg/compression.Bzip2.
// NOTE: Importing only this /types package does not inherently guarantee a Bzip2 algorithm
// will actually be available. (In fact it is intended for this types package not to depend
// on any of the implementations.)
const Bzip2AlgorithmName = "bzip2"

// Bzip2 compression.
var Bzip2 = NewAlgorithm(Bzip2AlgorithmName, Bzip2AlgorithmName,
	[]byte{0x42, 0x5A, 0x68}, bzip2Decompressor, bzip2Compressor)

func init() {
	Register(Bzip2)
}

// bzip2Decompressor is a DecompressorFunc for the bzip2 compression algorithm.
func bzip2Decompressor(r io.Reader) (io.ReadCloser, error) {
	return io.NopCloser(bzip2.NewReader(r)), nil
}

// bzip2Compressor is a CompressorFunc for the bzip2 compression algorithm.
func bzip2Compressor(r io.Writer, metadata map[string]string, level *int) (io.WriteCloser, error) {
	return nil, fmt.Errorf("bzip2 compression not supported")
}
