// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package compression

import (
	"fmt"
	"io"
	"sync"

	"github.com/pkg/errors"
)

var (
	lock                  sync.RWMutex
	compressionAlgorithms = map[string]Algorithm{}
)

func Register(algo Algorithm) {
	lock.Lock()
	defer lock.Unlock()
	compressionAlgorithms[algo.Name()] = algo
}

func noneDecompressor(r io.Reader) (io.ReadCloser, error) {
	return io.NopCloser(r), nil
}

func noneCompressor(w io.Writer, _ map[string]string, _ *int) (io.WriteCloser, error) {
	return NopWriteCloser(w), nil
}

var None = NewAlgorithm("none", "", nil, noneDecompressor, noneCompressor)

// AlgorithmByName returns the compressor by its name.
func AlgorithmByName(name string) (Algorithm, error) {
	lock.RLock()
	defer lock.RUnlock()

	algorithm, ok := compressionAlgorithms[name]
	if ok {
		return algorithm, nil
	}
	return nil, fmt.Errorf("cannot find compression algorithm %q", name)
}

// DetectCompression returns an Algorithm  if the input is recognized as a compressed format, an invalid
// value and nil otherwise.
// Because it consumes the start of input, other consumers must use the returned io.Reader instead to also read from the beginning.
func DetectCompression(input io.Reader) (Algorithm, io.Reader, error) {
	lock.RLock()
	defer lock.RUnlock()

	match := NewMatchReader(input)
	for _, algo := range compressionAlgorithms {
		match.Reset()
		ok, err := algo.Match(match)
		if err != nil {
			return nil, match.Reader(), err
		}
		if ok {
			return algo, match.Reader(), err
		}
	}
	return None, match.Reader(), nil
}

// AutoDecompress takes a stream and returns an uncompressed version of the
// same stream.
// The caller must call Close() on the returned stream (even if the input does not need,
// or does not even support, closing!).
func AutoDecompress(stream io.Reader) (io.ReadCloser, bool, error) {
	algo, stream, err := DetectCompression(stream)
	if err != nil {
		return nil, false, errors.Wrapf(err, "detecting compression")
	}
	res, err := algo.Decompressor(stream)
	if err != nil {
		return nil, false, errors.Wrapf(err, "initializing decompression")
	}
	return res, algo != None, nil
}
