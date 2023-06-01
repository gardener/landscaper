// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package compression

import (
	"io"

	"github.com/ulikunitz/xz"
)

// XzAlgorithmName is the name used by pkg/compression.Xz.
// NOTE: Importing only this /types package does not inherently guarantee a Xz algorithm
// will actually be available. (In fact it is intended for this types package not to depend
// on any of the implementations.)
const XzAlgorithmName = "Xz"

// Xz compression.
var Xz = NewAlgorithm(XzAlgorithmName, XzAlgorithmName,
	[]byte{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00}, xzDecompressor, xzCompressor)

func init() {
	Register(Xz)
}

// xzDecompressor is a DecompressorFunc for the xz compression algorithm.
func xzDecompressor(r io.Reader) (io.ReadCloser, error) {
	r, err := xz.NewReader(r)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(r), nil
}

// xzCompressor is a CompressorFunc for the xz compression algorithm.
func xzCompressor(r io.Writer, metadata map[string]string, level *int) (io.WriteCloser, error) {
	return xz.NewWriter(r)
}
