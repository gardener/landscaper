// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package compression

import (
	"io"

	"github.com/klauspost/compress/zstd"
)

// ZstdAlgorithmName is the name used by pkg/compression.Zstd.
// NOTE: Importing only this /types package does not inherently guarantee a Zstd algorithm
// will actually be available. (In fact it is intended for this types package not to depend
// on any of the implementations.)
const ZstdAlgorithmName = "zstd"

// Zstd compression.
var Zstd = NewAlgorithm(ZstdAlgorithmName, ZstdAlgorithmName,
	[]byte{0x28, 0xb5, 0x2f, 0xfd}, ZstdDecompressor, zstdCompressor)

func init() {
	Register(Zstd)
}

type wrapperZstdDecoder struct {
	decoder *zstd.Decoder
}

func (w *wrapperZstdDecoder) Close() error {
	w.decoder.Close()
	return nil
}

func (w *wrapperZstdDecoder) DecodeAll(input, dst []byte) ([]byte, error) {
	return w.decoder.DecodeAll(input, dst)
}

func (w *wrapperZstdDecoder) Read(p []byte) (int, error) {
	return w.decoder.Read(p)
}

func (w *wrapperZstdDecoder) Reset(r io.Reader) error {
	return w.decoder.Reset(r)
}

func (w *wrapperZstdDecoder) WriteTo(wr io.Writer) (int64, error) {
	return w.decoder.WriteTo(wr)
}

func zstdReader(buf io.Reader) (io.ReadCloser, error) {
	decoder, err := zstd.NewReader(buf)
	return &wrapperZstdDecoder{decoder: decoder}, err
}

func zstdWriter(dest io.Writer) (io.WriteCloser, error) {
	return zstd.NewWriter(dest)
}

func zstdWriterWithLevel(dest io.Writer, level int) (*zstd.Encoder, error) {
	el := zstd.EncoderLevelFromZstd(level)
	return zstd.NewWriter(dest, zstd.WithEncoderLevel(el))
}

// zstdCompressor is a CompressorFunc for the zstd compression algorithm.
func zstdCompressor(r io.Writer, metadata map[string]string, level *int) (io.WriteCloser, error) {
	if level == nil {
		return zstdWriter(r)
	}
	return zstdWriterWithLevel(r, *level)
}

// ZstdDecompressor is a DecompressorFunc for the zstd compression algorithm.
func ZstdDecompressor(r io.Reader) (io.ReadCloser, error) {
	return zstdReader(r)
}
