// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testutils

import (
	"io"

	"github.com/gardener/landscaper/pkg/components/cnudie"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

func newTestResource(res *types.Resource, blobResolver model.BlobResolver) model.Resource {
	return cnudie.NewResource(res, blobResolver)
}

func NewTestResourceFromReader(res *types.Resource, reader io.Reader, info *types.BlobInfo) model.Resource {
	blobResolver := newTestBlobResolverFromReader(reader, info)
	return cnudie.NewResource(res, blobResolver)
}
