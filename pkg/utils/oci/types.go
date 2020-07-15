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
	"net/http"

	"github.com/containerd/containerd/remotes"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/landscaper/pkg/apis/config"
	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

type Client interface {
	// GetManifest returns the ocispec Manifest for a reference
	GetManifest(ctx context.Context, ref string) (*ocispecv1.Manifest, error)

	// Fetch fetches the blob for the given ocispec Descriptor.
	Fetch(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer) error
}

// Resolver is a interface that should return a new resolver if called.
type Resolver interface {
	// Resolver returns a new authenticated resolver.
	Resolver(ctx context.Context, client *http.Client, plainHTTP bool) (remotes.Resolver, error)
}

// Options contains all client options to configure the oci client.
type Options struct {
	// Paths configures local paths to search for docker configuration files
	Paths []string

	// Resolver sets the used resolver.
	// A default resolver will be created if not given.
	Resolver Resolver

	// CacheConfig contains the cache configuration.
	// Tis configuration will automatically create a new cache based on that configuration.
	// This cache can be overwritten with the Cache property.
	CacheConfig *config.OCICacheConfiguration

	// Cache is the oci cache to be used by the client
	Cache cache.Cache
}

// Option is the interface to specify different cache options
type Option interface {
	ApplyOption(options *Options)
}

// ApplyOptions applies the given list options on these options,
// and then returns itself (for convenient chaining).
func (o *Options) ApplyOptions(opts []Option) *Options {
	for _, opt := range opts {
		opt.ApplyOption(o)
	}
	return o
}

// WithCache configures a cache for the oci client
type WithCache struct {
	cache.Cache
}

func (c WithCache) ApplyOption(options *Options) {
	options.Cache = c.Cache
}

// WithResolver configures a resolver for the oci client
type WithResolver struct {
	Resolver
}

func (c WithResolver) ApplyOption(options *Options) {
	options.Resolver = c.Resolver
}

// WithConfiguration applies external oci configuration as internal options.
func WithConfiguration(cfg *config.OCIConfiguration) *WithConfigurationStruct {
	if cfg == nil {
		return nil
	}
	wc := WithConfigurationStruct(*cfg)
	return &wc
}

// WithConfiguration applies external oci configuration as internal options.
type WithConfigurationStruct config.OCIConfiguration

func (c *WithConfigurationStruct) ApplyOption(options *Options) {
	if c == nil {
		return
	}
	if len(c.ConfigFiles) != 0 {
		options.Paths = c.ConfigFiles
	}
	if c.Cache != nil {
		options.CacheConfig = c.Cache
	}
}
