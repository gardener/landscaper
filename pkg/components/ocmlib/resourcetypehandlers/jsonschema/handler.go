package jsonschema

import (
	"bytes"
	"context"
	"fmt"

	"github.com/open-component-model/ocm/pkg/common/compression"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/finalizer"

	"github.com/gardener/landscaper/apis/mediatype"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/ocmlib/registries"
)

func init() {
	registries.Registry.Register(mediatype.JSONSchemaType, New())
}

type SchemaHandler struct{}

func New() *SchemaHandler {
	return &SchemaHandler{}
}

func (h *SchemaHandler) GetResourceContent(ctx context.Context, r model.Resource, access ocm.ResourceAccess) (_ *model.TypedResourceContent, rerr error) {
	var finalize finalizer.Finalizer
	defer finalize.FinalizeWithErrorPropagationf(&rerr, "accessing (and decompressing) json schema")

	m, err := access.AccessMethod()
	if err != nil {
		return nil, err
	}
	finalize.Close(m)

	mimeType := m.MimeType()
	mt, err := mediatype.Parse(mimeType)
	if err != nil {
		return nil, fmt.Errorf("unable to parse media type %q: %w", mimeType, err)
	}
	if mt.Type != mediatype.JSONSchemaArtifactsMediaTypeV1 {
		return nil, fmt.Errorf("unknown media type %s expected %s", mimeType, mediatype.JSONSchemaArtifactsMediaTypeV1)
	}

	schemaRaw, err := m.Reader()
	if err != nil {
		return nil, err
	}
	finalize.Close(schemaRaw)

	schema, _, err := compression.AutoDecompress(schemaRaw)
	if err != nil {
		return nil, err
	}
	finalize.Close(schema)

	var buf bytes.Buffer
	_, err = buf.ReadFrom(schema)
	if err != nil {
		return nil, err
	}

	return h.Prepare(ctx, buf.Bytes())
}

func (h *SchemaHandler) Prepare(ctx context.Context, data []byte) (*model.TypedResourceContent, error) {
	return &model.TypedResourceContent{
		Type:     mediatype.JSONSchemaType,
		Resource: data,
	}, nil
}
