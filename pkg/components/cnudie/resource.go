// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cnudie

import (
	"context"
	"io"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/pkg/components/model"
)

type Resource struct {
	resource     *cdv2.Resource
	blobResolver ctf.BlobResolver
}

var _ model.Resource = &Resource{}

func NewResource(res *cdv2.Resource, blobResolver ctf.BlobResolver) *Resource {
	return &Resource{
		resource:     res,
		blobResolver: blobResolver,
	}
}

func (r Resource) GetName() string {
	return r.resource.GetName()
}

func (r Resource) GetVersion() string {
	return r.resource.GetVersion()
}

func (r Resource) GetDescriptor(ctx context.Context) ([]byte, error) {
	// TODO component model: implement me
	panic("implement me")
}

func (r Resource) GetBlob(ctx context.Context, writer io.Writer) error {
	_, err := r.blobResolver.Resolve(ctx, *r.resource, writer)
	return err
}

func (r Resource) GetBlobInfo(ctx context.Context) (*model.BlobInfo, error) {
	info, err := r.blobResolver.Info(ctx, *r.resource)
	if err != nil {
		return nil, err
	}

	return &model.BlobInfo{
		MediaType: info.MediaType,
		Digest:    info.Digest,
	}, nil
}
