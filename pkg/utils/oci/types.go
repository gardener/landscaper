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

package oci

import (
	"context"
	"io"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

type Client interface {
	// GetManifest returns the ocispec Manifest for a reference
	GetManifest(ctx context.Context, ref string) (*ocispecv1.Manifest, error)

	// Fetch fetches the blob for the given ocispec Descriptor.
	Fetch(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer) error
}

// Options contains all client options to configure the oci client.
type Options struct {
	Cache cache.Cache
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

type WithCache struct {
	cache.Cache
}

func (c WithCache) ApplyToList(options *Options) {
	options.Cache = c.Cache
}
