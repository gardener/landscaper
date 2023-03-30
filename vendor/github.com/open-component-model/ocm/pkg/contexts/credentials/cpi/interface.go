// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

// This is the Context Provider Interface for credential providers

import (
	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/internal"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
)

const (
	KIND_CREDENTIALS = internal.KIND_CREDENTIALS
	KIND_REPOSITORY  = internal.KIND_REPOSITORY
)

const CONTEXT_TYPE = internal.CONTEXT_TYPE

type (
	Context                = internal.Context
	ContextProvider        = internal.ContextProvider
	Repository             = internal.Repository
	RepositoryType         = internal.RepositoryType
	Credentials            = internal.Credentials
	CredentialsSource      = internal.CredentialsSource
	CredentialsChain       = internal.CredentialsChain
	CredentialsSpec        = internal.CredentialsSpec
	RepositorySpec         = internal.RepositorySpec
	GenericRepositorySpec  = internal.GenericRepositorySpec
	GenericCredentialsSpec = internal.GenericCredentialsSpec
	DirectCredentials      = internal.DirectCredentials
)

type (
	ConsumerIdentity = internal.ConsumerIdentity
	IdentityMatcher  = internal.IdentityMatcher
)

var DefaultContext = internal.DefaultContext

func New(m ...datacontext.BuilderMode) Context {
	return internal.Builder{}.New(m...)
}

func NewGenericCredentialsSpec(name string, repospec *GenericRepositorySpec) *GenericCredentialsSpec {
	return internal.NewGenericCredentialsSpec(name, repospec)
}

func NewCredentialsSpec(name string, repospec RepositorySpec) CredentialsSpec {
	return internal.NewCredentialsSpec(name, repospec)
}

func ToGenericCredentialsSpec(spec CredentialsSpec) (*GenericCredentialsSpec, error) {
	return internal.ToGenericCredentialsSpec(spec)
}

func ToGenericRepositorySpec(spec RepositorySpec) (*GenericRepositorySpec, error) {
	return internal.ToGenericRepositorySpec(spec)
}

func RegisterRepositoryType(name string, atype RepositoryType) {
	internal.DefaultRepositoryTypeScheme.Register(name, atype)
}

func RegisterStandardIdentityMatcher(typ string, matcher IdentityMatcher, desc string) {
	internal.StandardIdentityMatchers.Register(typ, matcher, desc)
}

func NewCredentials(props common.Properties) Credentials {
	return internal.NewCredentials(props)
}

func ErrUnknownCredentials(name string) error {
	return internal.ErrUnknownCredentials(name)
}

func ErrUnknownRepository(kind, name string) error {
	return internal.ErrUnknownRepository(kind, name)
}

func CredentialsForConsumer(ctx ContextProvider, id ConsumerIdentity, matchers ...IdentityMatcher) (Credentials, error) {
	return internal.CredentialsForConsumer(ctx, id, false, matchers...)
}

func RequiredCredentialsForConsumer(ctx ContextProvider, id ConsumerIdentity, matchers ...IdentityMatcher) (Credentials, error) {
	return internal.CredentialsForConsumer(ctx, id, true, matchers...)
}

var (
	CompleteMatch = internal.CompleteMatch
	NoMatch       = internal.NoMatch
	PartialMatch  = internal.PartialMatch
)
