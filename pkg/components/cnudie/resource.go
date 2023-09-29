// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cnudie

import (
	"context"
	"fmt"
	"github.com/gardener/landscaper/pkg/components/ocmlib/registries"
	"io"

	"github.com/gardener/component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

func NewResource(res *types.Resource, blobResolver ctf.BlobResolver) model.Resource {
	return &Resource{
		resource:        res,
		blobResolver:    blobResolver,
		handlerRegistry: registries.Registry,
	}
}

type Resource struct {
	resource        *types.Resource
	blobResolver    ctf.BlobResolver
	handlerRegistry *registries.ResourceHandlerRegistry
}

var _ model.Resource = &Resource{}

func (r Resource) GetName() string {
	return r.resource.GetName()
}

func (r Resource) GetVersion() string {
	return r.resource.GetVersion()
}

func (r Resource) GetType() string {
	return r.resource.GetType()
}

func (r Resource) GetAccessType() string {
	return r.resource.Access.GetType()
}

func (r Resource) GetResource() (*types.Resource, error) {
	return r.resource, nil
}

func (r Resource) GetBlob(ctx context.Context, writer io.Writer) (*types.BlobInfo, error) {
	return r.blobResolver.Resolve(ctx, *r.resource, writer)
}

func (r Resource) GetBlobInfo(ctx context.Context) (*types.BlobInfo, error) {
	return r.blobResolver.Info(ctx, *r.resource)
}

func (r *Resource) GetTypedContent(ctx context.Context) (*model.TypedResourceContent, error) {
	handler := r.handlerRegistry.Get(r.GetType())
	if handler != nil {
		// TODO
		// return handler.GetResourceContent(ctx, r, r.blobResolver)
		return nil, model.NotImplemented()
	}
	return nil, fmt.Errorf("no handler found for resource type %s", r.GetType())
}

func (r *Resource) GetCachingIdentity(ctx context.Context) string {
	blobInfo, _ := r.blobResolver.Info(ctx, *r.resource)
	if blobInfo == nil {
		return ""
	}
	return blobInfo.Digest
}
