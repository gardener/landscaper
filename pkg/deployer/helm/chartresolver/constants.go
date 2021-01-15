// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package chartresolver

const (
	// HelmChartConfigMediaType is the reserved media type for the Helm chart manifest config
	HelmChartConfigMediaType = "application/vnd.cncf.helm.config.v1+json"

	// HelmChartContentLayerMediaType is the reserved media type for Helm chart package content
	HelmChartContentLayerMediaType = "application/tar+gzip"

	// OldHelmResourceType describes the old helm resource type of a component descrptor defined resource.
	OldHelmResourceType = "helm"
	// HelmChartResourceType describes the helm resource type of a component descrptor defined resource.
	HelmChartResourceType = "helm.io/chart"
)
