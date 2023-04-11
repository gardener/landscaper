// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package credentials

import (
	"context"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/internal"
	"github.com/open-component-model/ocm/pkg/contexts/oci/identity"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	KIND_CREDENTIALS = internal.KIND_CREDENTIALS
	KIND_CONSUMER    = internal.KIND_CONSUMER
	KIND_REPOSITORY  = internal.KIND_REPOSITORY
)

const CONTEXT_TYPE = internal.CONTEXT_TYPE

const AliasRepositoryType = internal.AliasRepositoryType

type (
	Context              = internal.Context
	ContextProvider      = internal.ContextProvider
	RepositoryTypeScheme = internal.RepositoryTypeScheme
	Repository           = internal.Repository
	Credentials          = internal.Credentials
	CredentialsSource    = internal.CredentialsSource
	CredentialsChain     = internal.CredentialsChain
	CredentialsSpec      = internal.CredentialsSpec
	RepositorySpec       = internal.RepositorySpec
)

type (
	ConsumerIdentity        = internal.ConsumerIdentity
	IdentityMatcher         = internal.IdentityMatcher
	IdentityMatcherInfo     = internal.IdentityMatcherInfo
	IdentityMatcherRegistry = internal.IdentityMatcherRegistry
)

type (
	GenericRepositorySpec  = internal.GenericRepositorySpec
	GenericCredentialsSpec = internal.GenericCredentialsSpec
	DirectCredentials      = internal.DirectCredentials
)

func DefaultContext() internal.Context {
	return internal.DefaultContext
}

func ForContext(ctx context.Context) Context {
	return internal.ForContext(ctx)
}

func DefinedForContext(ctx context.Context) (Context, bool) {
	return internal.DefinedForContext(ctx)
}

func NewCredentialsSpec(name string, repospec RepositorySpec) CredentialsSpec {
	return internal.NewCredentialsSpec(name, repospec)
}

func NewGenericCredentialsSpec(name string, repospec *GenericRepositorySpec) CredentialsSpec {
	return internal.NewGenericCredentialsSpec(name, repospec)
}

func NewGenericRepositorySpec(data []byte, unmarshaler runtime.Unmarshaler) (RepositorySpec, error) {
	return internal.NewGenericRepositorySpec(data, unmarshaler)
}

func NewCredentials(props common.Properties) Credentials {
	return internal.NewCredentials(props)
}

func ToGenericCredentialsSpec(spec CredentialsSpec) (*GenericCredentialsSpec, error) {
	return internal.ToGenericCredentialsSpec(spec)
}

func ToGenericRepositorySpec(spec RepositorySpec) (*GenericRepositorySpec, error) {
	return internal.ToGenericRepositorySpec(spec)
}

func ErrUnknownCredentials(name string) error {
	return internal.ErrUnknownCredentials(name)
}

// CredentialsForConsumer determine effective credentials for a consumer.
// If no credentials are configured no error and nil is returned.
// It evaluates a found credentials source for the consumer to determine the
// final credential properties.
func CredentialsForConsumer(ctx ContextProvider, id ConsumerIdentity, matchers ...IdentityMatcher) (Credentials, error) {
	return internal.CredentialsForConsumer(ctx, id, false, matchers...)
}

// RequiredCredentialsForConsumer like CredentialsForConsumer, but an errors is returned
// if no credentials are found.
func RequiredCredentialsForConsumer(ctx ContextProvider, id ConsumerIdentity, matchers ...IdentityMatcher) (Credentials, error) {
	return internal.CredentialsForConsumer(ctx, id, true, matchers...)
}

var (
	CompleteMatch = internal.CompleteMatch
	NoMatch       = internal.NoMatch
	PartialMatch  = internal.PartialMatch
)

func NewConsumerIdentity(typ string, attrs ...string) ConsumerIdentity {
	r := map[string]string{}
	r[identity.ID_TYPE] = typ

	i := 0
	for len(attrs) > i {
		r[attrs[i]] = r[attrs[i+1]]
		i += 2
	}
	return r
}
