// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type inmemoryCache struct {
	store map[string][]byte
}

// NewInMemoryCache creates a new in memory cache.
func NewInMemoryCache() *inmemoryCache {
	return &inmemoryCache{
		store: make(map[string][]byte),
	}
}

func (fs *inmemoryCache) Close() error {
	return nil
}

func (fs *inmemoryCache) Get(desc ocispecv1.Descriptor) (io.ReadCloser, error) {
	data, ok := fs.store[desc.Digest.String()]
	if !ok {
		return nil, ErrNotFound
	}
	return ioutil.NopCloser(bytes.NewBuffer(data)), nil
}

func (fs *inmemoryCache) Add(desc ocispecv1.Descriptor, reader io.ReadCloser) error {
	if _, ok := fs.store[desc.Digest.String()]; ok {
		// already cached
		return nil
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return fmt.Errorf("unable to read data: %w", err)
	}
	fs.store[desc.Digest.String()] = buf.Bytes()
	return nil
}
