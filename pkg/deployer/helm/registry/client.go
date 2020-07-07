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
	"github.com/deislabs/oras/pkg/auth"
	dockerauth "github.com/deislabs/oras/pkg/auth/docker"
	orascontent "github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"
	"helm.sh/helm/v3/pkg/chart"
	chartloader "helm.sh/helm/v3/pkg/chart/loader"

	"github.com/gardener/landscaper/pkg/utils/oci"
)

type Client struct {
	authorizer auth.Client
	resolver   remotes.Resolver
}

// NewClient creates a new helm oci registry client.
func NewClient(configFiles ...string) (*Client, error) {
	authorizer, err := dockerauth.NewClient(configFiles...)
	if err != nil {
		return nil, err
	}

	resolver, err := authorizer.Resolver(context.Background(), http.DefaultClient, false)
	if err != nil {
		return nil, err
	}
	return &Client{
		authorizer: authorizer,
		resolver:   resolver,
	}, nil
}

// GetChart pulls a chart from a oci registry with the given ref
func (c *Client) GetChart(ctx context.Context, ref string) (*chart.Chart, error) {
	ingester := orascontent.NewMemoryStore()
	desc, _, err := oras.Pull(ctx, c.resolver, ref, ingester,
		oras.WithPullEmptyNameAllowed(),
		oras.WithAllowedMediaTypes(KnownMediaTypes()),
		oras.WithContentProvideIngester(ingester))
	if err != nil {
		return nil, err
	}

	manifest, err := oci.ParseManifest(ingester, desc)
	if err != nil {
		return nil, err
	}

	if len(manifest.Layers) != 1 {
		return nil, errors.New("unexpected number of layers")
	}

	if manifest.Layers[0].MediaType != HelmChartContentLayerMediaType {
		return nil, errors.New("unexpected media type of content")
	}

	_, blob, ok := ingester.Get(manifest.Layers[0])
	if !ok {
		return nil, errors.New("no blob found")
	}

	return chartloader.LoadArchive(bytes.NewBuffer(blob))
}
