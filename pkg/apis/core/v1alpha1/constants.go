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

	// BlueprintType is the name of the blueprint type in a component descriptor.
	BlueprintType = "landscaper.gardener.cloud/blueprint"

	// OldBlueprintType is the old name of the blueprint type in a component descriptor.
	OldBlueprintType = "blueprint"

	// BlueprintFileName is the filename of a component definition on a local path
	BlueprintFileName = "blueprint.yaml"

	// BlueprintArtifactsMediaType is the reserved media type for a blueprint that is tored as its own artifact.
	BlueprintArtifactsMediaType = "application/vnd.gardener.landscaper.blueprint.v1+tar+gzip"
)
