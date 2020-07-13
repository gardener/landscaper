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

package registry

import (
	"bytes"
	"context"
	"errors"
	"net/http"

	"github.com/containerd/containerd/remotes"
	dockerauth "github.com/deislabs/oras/pkg/auth/docker"
	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/chart"
	chartloader "helm.sh/helm/v3/pkg/chart/loader"

	helmv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/pkg/utils/oci"
	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

type Client struct {
	oci oci.Client
}

// NewClient creates a new helm oci registry client.
func NewClient(log logr.Logger, config *helmv1alpha1.Configuration) (*Client, error) {
	authorizer, err := dockerauth.NewClient()
	if err != nil {
		return nil, err
	}

	resolver, err := authorizer.Resolver(context.Background(), http.DefaultClient, false)
	if err != nil {
		return nil, err
	}

	ociClient, err := buildOCIClient(log, resolver, config)
	if err != nil {
		return nil, err
	}
	return &Client{
		oci: ociClient,
	}, nil
}

// GetChart pulls a chart from a oci registry with the given ref
func (c *Client) GetChart(ctx context.Context, ref string) (*chart.Chart, error) {
	manifest, err := c.oci.GetManifest(ctx, ref)
	if err != nil {
		return nil, err
	}

	if len(manifest.Layers) != 1 {
		return nil, errors.New("unexpected number of layers")
	}

	if manifest.Layers[0].MediaType != HelmChartContentLayerMediaType {
		return nil, errors.New("unexpected media type of content")
	}

	var data bytes.Buffer
	if err := c.oci.Fetch(ctx, ref, manifest.Layers[0], &data); err != nil {
		return nil, err
	}

	return chartloader.LoadArchive(&data)
}

func buildOCIClient(log logr.Logger, resolver remotes.Resolver, config *helmv1alpha1.Configuration) (oci.Client, error) {
	opts := make([]oci.Option, 0)
	if config.OCICache != nil {
		ocicache, err := cache.NewCache(log, applyCacheConfigs(config)...)
		if err != nil {
			return nil, err
		}
		opts = append(opts, oci.WithCache{Cache: ocicache})
	}
	return oci.NewClient(log, resolver, opts...)
}

func applyCacheConfigs(config *helmv1alpha1.Configuration) []cache.Option {
	opts := make([]cache.Option, 0)
	if config.OCICache.UseInMemoryOverlay {
		opts = append(opts, cache.WithInMemoryOverlay(true))
	}
	if len(config.OCICache.Path) != 0 {
		opts = append(opts, cache.WithBasePath(config.OCICache.Path))
	}
	return opts
}
