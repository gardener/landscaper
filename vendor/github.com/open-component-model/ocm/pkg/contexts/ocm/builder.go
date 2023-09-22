// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"context"

	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/internal"
)

func WithContext(ctx context.Context) internal.Builder {
	return internal.Builder{}.WithContext(ctx)
}

func WithCredentials(ctx credentials.Context) internal.Builder {
	return internal.Builder{}.WithCredentials(ctx)
}

func WithOCIRepositories(ctx oci.Context) internal.Builder {
	return internal.Builder{}.WithOCIRepositories(ctx)
}

func WithRepositoyTypeScheme(scheme RepositoryTypeScheme) internal.Builder {
	return internal.Builder{}.WithRepositoyTypeScheme(scheme)
}

func WithRepositoryDelegation(reg RepositoryDelegationRegistry) internal.Builder {
	return internal.Builder{}.WithRepositoryDelegation(reg)
}

func WithAccessypeScheme(scheme AccessTypeScheme) internal.Builder {
	return internal.Builder{}.WithAccessTypeScheme(scheme)
}

func WithRepositorySpecHandlers(reg RepositorySpecHandlers) internal.Builder {
	return internal.Builder{}.WithRepositorySpecHandlers(reg)
}

func WithBlobHandlers(reg BlobHandlerRegistry) internal.Builder {
	return internal.Builder{}.WithBlobHandlers(reg)
}

func WithLabelMergeHandlers(reg LabelMergeHandlerRegistry) internal.Builder {
	return internal.Builder{}.WithLabelMergeHandlers(reg)
}

func WithBlobDigesters(reg BlobDigesterRegistry) internal.Builder {
	return internal.Builder{}.WithBlobDigesters(reg)
}

func New(mode ...datacontext.BuilderMode) Context {
	return internal.Builder{}.New(mode...)
}
