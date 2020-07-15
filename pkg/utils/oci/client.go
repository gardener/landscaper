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
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	dockerauth "github.com/deislabs/oras/pkg/auth/docker"
	"github.com/go-logr/logr"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

type client struct {
	log      logr.Logger
	resolver Resolver
	cache    cache.Cache
}

// NewClient creates a new OCI Client.
func NewClient(log logr.Logger, opts ...Option) (Client, error) {
	options := &Options{}
	options.ApplyOptions(opts)

	if options.Resolver == nil {
		authorizer, err := dockerauth.NewClient(options.Paths...)
		if err != nil {
			return nil, err
		}
		options.Resolver = authorizer
	}

	if options.Cache == nil && options.CacheConfig != nil {
		c, err := cache.NewCache(log, cache.WithConfiguration(options.CacheConfig))
		if err != nil {
			return nil, err
		}
		options.Cache = c
	}

	return &client{
		log:      log,
		resolver: options.Resolver,
		cache:    options.Cache,
	}, nil
}

func (c *client) GetManifest(ctx context.Context, ref string) (*ocispecv1.Manifest, error) {
	resolver, err := c.resolver.Resolver(context.Background(), http.DefaultClient, false)
	if err != nil {
		return nil, err
	}
	_, desc, err := resolver.Resolve(ctx, ref)
	if err != nil {
		return nil, err
	}

	data := bytes.NewBuffer([]byte{})
	if err := c.Fetch(ctx, ref, desc, data); err != nil {
		return nil, err
	}

	var manifest ocispecv1.Manifest
	if err := json.Unmarshal(data.Bytes(), &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func (c *client) Fetch(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer) error {
	reader, err := c.getFetchReader(ctx, ref, desc)
	if err != nil {
		return err
	}
	defer func() {
		if err := reader.Close(); err != nil {
			c.log.Error(err, "failed closing reader", "ref", ref)
		}
	}()

	if _, err := io.Copy(writer, reader); err != nil {
		return err
	}
	return nil
}

func (c *client) getFetchReader(ctx context.Context, ref string, desc ocispecv1.Descriptor) (io.ReadCloser, error) {
	if c.cache != nil {
		reader, err := c.cache.Get(desc)
		if err != nil && err != cache.ErrNotFound {
			return nil, err
		}
		if err == nil {
			return reader, nil
		}
	}

	resolver, err := c.resolver.Resolver(context.Background(), http.DefaultClient, false)
	if err != nil {
		return nil, err
	}
	fetcher, err := resolver.Fetcher(ctx, ref)
	if err != nil {
		return nil, err
	}
	reader, err := fetcher.Fetch(ctx, desc)
	if err != nil {
		return nil, err
	}
	// try to cache
	if c.cache != nil {
		if err := c.cache.Add(desc, reader); err != nil {
			// do not throw an error as cache is just an optimization
			c.log.V(5).Info("unable to cache descriptor", "ref", ref, "error", err.Error())
		}
		return c.cache.Get(desc)
	}

	return reader, err
}
