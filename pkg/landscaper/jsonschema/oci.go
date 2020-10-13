// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package jsonschema

import (
	"bytes"
	"context"
	"fmt"

	"github.com/gardener/landscaper/pkg/utils/oci"
)

// FetchFromOCIRegistry fetches a jsonschema from a oci registry.
func FetchFromOCIRegistry(ctx context.Context, ociClient oci.Client, ref string) ([]byte, error) {
	manifest, err := ociClient.GetManifest(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve jsonschema from %s for: %w", ref, err)
	}
	if len(manifest.Layers) != 1 {
		return nil, fmt.Errorf("expected exactly one layer that contains the jsonschema but got %d in '%s'", len(manifest.Layers), ref)
	}
	if manifest.Layers[0].MediaType != JSONSchemaMediaType {
		return nil, fmt.Errorf("expected the oci descptor layer to be '%s' but got '%s'", JSONSchemaMediaType, manifest.Layers[0].MediaType)
	}
	// the first layer is expected to contain al valid jsonschema
	var JSONSchemaBytes bytes.Buffer
	if err := ociClient.Fetch(ctx, ref, manifest.Layers[0], &JSONSchemaBytes); err != nil {
		return nil, fmt.Errorf("unable to fetch jsonschema '%s' from oci registry: %w", ref, err)
	}
	return JSONSchemaBytes.Bytes(), nil
}
