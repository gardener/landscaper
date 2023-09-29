// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"github.com/open-component-model/ocm/pkg/contexts/credentials/internal"
)

const AliasRepositoryType = internal.AliasRepositoryType

type AliasRegistry = internal.AliasRegistry

type aliasRegistry struct {
	RepositoryType
	setter internal.SetAliasFunction
}

var _ AliasRegistry = &aliasRegistry{}

func NewAliasRegistry(t RepositoryType, setter internal.SetAliasFunction) RepositoryType {
	return &aliasRegistry{
		RepositoryType: t,
		setter:         setter,
	}
}

func (a *aliasRegistry) SetAlias(ctx Context, name string, spec RepositorySpec, creds CredentialsSource) error {
	return a.setter(ctx, name, spec, creds)
}
