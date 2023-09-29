// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

const AliasRepositoryType = "Alias"

type SetAliasFunction func(ctx Context, name string, spec RepositorySpec, creds CredentialsSource) error

type AliasRegistry interface {
	SetAlias(ctx Context, name string, spec RepositorySpec, creds CredentialsSource) error
}
