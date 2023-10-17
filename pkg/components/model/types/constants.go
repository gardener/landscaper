// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package types

// Constants for resource types
const (
	AuthHeaderSecretDefaultKey = "authHeader"

	// OldHelmResourceType describes the old resource type of helm chart resources defined in a component descriptor.
	OldHelmResourceType = "helm"

	// HelmChartResourceType describes the resource type of helm chart resources defined in a component descriptor.
	HelmChartResourceType = "helm.io/chart"
)
