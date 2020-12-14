// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"bytes"
	"context"
	"errors"

	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/chart"
	chartloader "helm.sh/helm/v3/pkg/chart/loader"

	"github.com/gardener/component-cli/ociclient"

	confighelper "github.com/gardener/landscaper/pkg/apis/config/helper"
	helmv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/helm/v1alpha1"
)

type Client struct {
	oci ociclient.Client
}

// NewClient creates a new helm oci registry client.
func NewClient(log logr.Logger, config *helmv1alpha1.Configuration) (*Client, error) {
	ociClient, err := ociclient.NewClient(log, confighelper.WithConfiguration(config.OCI))
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
