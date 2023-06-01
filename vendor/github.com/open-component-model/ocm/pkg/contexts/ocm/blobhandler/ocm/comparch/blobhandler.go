// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package comparch

import (
	"fmt"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localfsblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/compatattr"
	storagecontext "github.com/open-component-model/ocm/pkg/contexts/ocm/blobhandler/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/comparch"
	"github.com/open-component-model/ocm/pkg/errors"
)

func init() {
	cpi.RegisterBlobHandler(NewBlobHandler(), cpi.ForRepo(cpi.CONTEXT_TYPE, comparch.Type))
}

////////////////////////////////////////////////////////////////////////////////

// blobHandler is the default handling to store local blobs as local blobs.
type blobHandler struct{}

func NewBlobHandler() cpi.BlobHandler {
	return &blobHandler{}
}

func (b *blobHandler) StoreBlob(blob cpi.BlobAccess, artType, hint string, global cpi.AccessSpec, ctx cpi.StorageContext) (cpi.AccessSpec, error) {
	ocmctx, ok := ctx.(storagecontext.StorageContext)
	if !ok {
		return nil, fmt.Errorf("failed to assert type %T to storagecontext.StorageContext", ctx)
	}

	if blob == nil {
		return nil, errors.New("a resource has to be defined")
	}
	err := ocmctx.AddBlob(blob)
	if err != nil {
		return nil, err
	}
	path := common.DigestToFileName(blob.Digest())
	if compatattr.Get(ctx.GetContext()) {
		return localfsblob.New(path, blob.MimeType()), nil
	} else {
		return localblob.New(path, hint, blob.MimeType(), global), nil
	}
}
