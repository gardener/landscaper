// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"context"
	"io"

	"github.com/gardener/landscaper/pkg/components/model/types"
)

type Resource interface {

	// GetName returns the name by which the resource can be identified among all resources of a component version.
	GetName() string

	// GetVersion is a design error.
	GetVersion() string

	// GetType returns the type of the resource. It indicates whether the resource is for example a blueprint,
	// helm chart, or json schema. (Not to be confused with the access type.)
	GetType() string

	// GetAccessType returns the access type of the resource, for example: "localOciBlob" (cdv2.LocalOCIBlobType)
	GetAccessType() string

	// GetResource returns the entry in the component descriptor that corresponds to the present resource.
	GetResource() (*types.Resource, error)

	// GetBlob returns the content of the resource, written to the given Writer.
	GetBlob(ctx context.Context, writer io.Writer) (*types.BlobInfo, error)

	// GetBlobInfo returns information like mediatype and digest of the resource.
	GetBlobInfo(ctx context.Context) (*types.BlobInfo, error)
}
