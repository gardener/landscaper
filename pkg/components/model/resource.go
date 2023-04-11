// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"context"
	"io"
)

type Resource interface {
	GetName() string
	GetVersion() string
	GetDescriptor(ctx context.Context) ([]byte, error)
	GetBlob(ctx context.Context, writer io.Writer) error
	GetBlobInfo(ctx context.Context) (*BlobInfo, error)
}

type BlobInfo struct {
	MediaType string
	Digest    string
}
