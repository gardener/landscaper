// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cache

import (
	"errors"
	"io"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

var (
	// ErrNotFound is a error that indicates that the file is not cached
	ErrNotFound = errors.New("not cached")
)

// Cache is the interface for a oci cache
type Cache interface {
	Get(desc ocispecv1.Descriptor) (io.ReadCloser, error)
	Add(desc ocispecv1.Descriptor, reader io.ReadCloser) error
}

// Options contains all oci cache options to configure the oci cache.
type Options struct {
	// InMemoryOverlay specifies if a overlayFs InMemory cache should be used
	InMemoryOverlay bool

	// BasePath specifies the Base path for the os filepath.
	// Will be defaulted to a temp filepath if not specified
	BasePath string
}

// Option is the interface to specify different cache options
type Option interface {
	ApplyToList(options *Options)
}

// ApplyOptions applies the given list options on these options,
// and then returns itself (for convenient chaining).
func (o *Options) ApplyOptions(opts []Option) *Options {
	for _, opt := range opts {
		opt.ApplyToList(o)
	}
	return o
}

// WithInMemoryOverlay is the options to specify the usage of a in memory overlayFs
type WithInMemoryOverlay bool

func (w WithInMemoryOverlay) ApplyToList(options *Options) {
	options.InMemoryOverlay = bool(w)
}

// WithBasePath is the options to specify a base path
type WithBasePath string

func (p WithBasePath) ApplyToList(options *Options) {
	options.BasePath = string(p)
}
