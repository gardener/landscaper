// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package mediatype

const (
	// BlueprintType is the name of the blueprint type in a component descriptor.
	BlueprintType = "landscaper.gardener.cloud/blueprint"

	// OldBlueprintType is the old name of the blueprint type in a component descriptor.
	OldBlueprintType = "blueprint"

	// BlueprintArtifactsMediaTypeV0 is the reserved media type for a blueprint that is stored as its own artifact.
	// This is the legacy deprecated artifact media type that was used for the layer and the config type.
	// Use BlueprintArtifactsConfigMediaTypeV1 or BlueprintArtifactsLayerMediaTypeV1 instead.
	// DEPRECATED
	BlueprintArtifactsMediaTypeV0 = "application/vnd.gardener.landscaper.blueprint.v1+tar+gzip"

	// BlueprintArtifactsConfigMediaTypeV1 is the config reserved media type for a blueprint that is stored as its own artifact.
	// This describes the config media type of the blueprint artifact.
	// The suffix can be yaml or json whereas yaml is the default.
	BlueprintArtifactsConfigMediaTypeV1 = "application/vnd.gardener.landscaper.blueprint.config.v1"

	// BlueprintArtifactsLayerMediaTypeV1 is the reserved layer media type for a blueprint that is stored as its own artifact.
	// This describes the layer media type of the blueprint artifact as well as the media type of a localOciBlob.
	// Optionally the compression can be added as "+gzip"
	BlueprintArtifactsLayerMediaTypeV1 = "application/vnd.gardener.landscaper.blueprint.layer.v1.tar"

	// JSONSchemaArtifactsMediaTypeV0 is the reserved media type for a jsonschema that is stored as layer in an oci artifact.
	// This is the legacy deprecated artifact media type use JSONSchemaArtifactsMediaTypeV1 instead.
	// DEPRECATED
	JSONSchemaArtifactsMediaTypeV0 = "application/vnd.gardener.landscaper.jsonscheme.v1+json"

	// JSONSchemaArtifactsMediaTypeV1 is the reserved media type for a jsonschema that is stored as layer in an oci artifact.
	// This is the legacy deprecated artifact media type.
	JSONSchemaArtifactsMediaTypeV1 = "application/vnd.gardener.landscaper.jsonschema.layer.v1.json"

	// GZipCompression is the identifier for a gzip compressed file.
	GZipCompression = "gzip"

	// MediaTypeGZip defines the media type for a gzipped file
	MediaTypeGZip = "application/gzip"
)

// DefaultMediaTypeConversions defines the default conversions for landscaper media types.
func DefaultMediaTypeConversions(mediaType string) (convertedType string, converted bool, err error) {
	switch mediaType {
	case BlueprintArtifactsMediaTypeV0:
		return NewBuilder(BlueprintArtifactsLayerMediaTypeV1).Compression(GZipCompression).String(), true, nil
	case JSONSchemaArtifactsMediaTypeV0:
		return JSONSchemaArtifactsMediaTypeV1, true, nil
	}
	return "", false, nil
}
