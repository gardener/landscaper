// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package jsonschema

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"

	"github.com/gardener/landscaper/apis/mediatype"
	"github.com/gardener/landscaper/pkg/components/cnudie/registries"
	"github.com/gardener/landscaper/pkg/components/model"
)

func init() {
	registries.Registry.Register(mediatype.JSONSchemaType, New())
}

type SchemaHandler struct{}

func New() *SchemaHandler {
	return &SchemaHandler{}
}

func (h *SchemaHandler) GetResourceContent(ctx context.Context, r model.Resource, blobResolver model.BlobResolver) (_ *model.TypedResourceContent, rerr error) {
	var JSONSchemaBuf bytes.Buffer
	resource, err := r.GetResource()
	if err != nil {
		return nil, err
	}
	info, err := blobResolver.Resolve(ctx, *resource, &JSONSchemaBuf)
	if err != nil {
		return nil, err
	}

	mt, err := mediatype.Parse(info.MediaType)
	if err != nil {
		return nil, fmt.Errorf("unable to parse media type %q: %w", info.MediaType, err)
	}
	if mt.Type != mediatype.JSONSchemaArtifactsMediaTypeV1 {
		return nil, fmt.Errorf("unknown media type %s expected %s", info.MediaType, mediatype.JSONSchemaArtifactsMediaTypeV1)
	}

	result := JSONSchemaBuf.Bytes()

	if mt.IsCompressed(mediatype.GZipCompression) {
		var decompJSONSchemaBuf bytes.Buffer
		r, err := gzip.NewReader(&JSONSchemaBuf)
		if err != nil {
			return nil, fmt.Errorf("unable to decompress jsonschema: %w", err)
		}
		if _, err := io.Copy(&decompJSONSchemaBuf, r); err != nil {
			return nil, fmt.Errorf("unable to decompress jsonschema: %w", err)
		}
		result = decompJSONSchemaBuf.Bytes()
	}

	return h.Prepare(ctx, result)
}

func (h *SchemaHandler) Prepare(ctx context.Context, data []byte) (_ *model.TypedResourceContent, rerr error) {
	return &model.TypedResourceContent{
		Type:     mediatype.JSONSchemaType,
		Resource: data,
	}, nil
}
