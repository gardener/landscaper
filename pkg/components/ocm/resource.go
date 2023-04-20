// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"context"
	"io"

	"github.com/open-component-model/ocm/pkg/contexts/ocm"

	"github.com/gardener/landscaper/pkg/components/model"
)

type Resource struct {
	resourceAccess ocm.ResourceAccess
}

var _ model.Resource = &Resource{}

func newResource(res ocm.ResourceAccess) *Resource {
	return &Resource{
		resourceAccess: res,
	}
}

func (r Resource) GetName() string {
	return r.resourceAccess.Meta().Name
}

func (r Resource) GetVersion() string {
	return r.resourceAccess.Meta().Version
}

func (r Resource) GetDescriptor(ctx context.Context) ([]byte, error) {
	panic("not possible to implement without pulling an arm out")
}

func (r Resource) GetBlob(ctx context.Context, writer io.Writer) error {
	meth, err := r.resourceAccess.AccessMethod()
	if err != nil {
		return err
	}
	defer meth.Close()

	data, err := meth.Get()
	if err != nil {
		return err
	}
	if _, err := writer.Write(data); err != nil {
		return err
	}
	return nil
}

func (r Resource) GetBlobInfo(ctx context.Context) (*model.BlobInfo, error) {
	digest := r.resourceAccess.Meta().Digest.String()
	meth, err := r.resourceAccess.AccessMethod()
	if err != nil {
		return nil, err
	}
	defer meth.Close()

	mediatype := meth.MimeType()

	return &model.BlobInfo{
		MediaType: mediatype,
		Digest:    digest,
	}, nil
}
