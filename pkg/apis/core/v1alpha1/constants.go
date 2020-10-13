// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

const (
	// LandscapeConfigName is the namespace unique name of the landscape configuration
	LandscapeConfigName = "default"

	// DataObjectSecretDataKey is the key of the secret where the landscape and installations stores their merged configuration.
	DataObjectSecretDataKey = "config"

	// LandscaperFinalizer is the finalizer of the landscaper
	LandscaperFinalizer = "finalizer.landscaper.gardener.cloud"

	// Annotations

	// OperationAnnotation is the annotation that specifies a operation for a component
	OperationAnnotation = "landscaper.gardener.cloud/operation"

	// BlueprintFilePath is the path to the component definition
	BlueprintFilePath = "/blueprint.yaml"

	// ComponentDefinitionComponentDescriptorPath is the path to the component descriptor
	ComponentDefinitionComponentDescriptorPath = "component_descriptor.yaml"
)
