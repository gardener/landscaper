// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"context"
	"io"

	"github.com/gardener/landscaper/pkg/components/model/types"
)

type BlobResolver interface {
	Info(ctx context.Context, res types.Resource) (*types.BlobInfo, error)

	Resolve(ctx context.Context, res types.Resource, writer io.Writer) (*types.BlobInfo, error)
}
