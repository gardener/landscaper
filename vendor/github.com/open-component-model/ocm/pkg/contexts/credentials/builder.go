// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package credentials

import (
	"context"

	"github.com/open-component-model/ocm/pkg/contexts/config"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/internal"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
)

func WithContext(ctx context.Context) internal.Builder {
	return internal.Builder{}.WithContext(ctx)
}

func WithConfigs(ctx config.Context) internal.Builder {
	return internal.Builder{}.WithConfig(ctx)
}

func WithRepositoyTypeScheme(scheme RepositoryTypeScheme) internal.Builder {
	return internal.Builder{}.WithRepositoyTypeScheme(scheme)
}

func WithStandardConumerMatchers(matchers internal.IdentityMatcherRegistry) internal.Builder {
	return internal.Builder{}.WithStandardConumerMatchers(matchers)
}

func New(mode ...datacontext.BuilderMode) Context {
	return internal.Builder{}.New(mode...)
}
