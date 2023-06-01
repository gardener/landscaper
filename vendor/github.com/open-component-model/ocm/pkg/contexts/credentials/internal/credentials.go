// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"github.com/modern-go/reflect2"
)

// CredentialsSource is a factory for effective credentials.
type CredentialsSource interface {
	Credentials(Context, ...CredentialsSource) (Credentials, error)
}

// CredentialsChain is a chain of credentials, where the
// credential i+1 (is present) is used to resolve credential i.
type CredentialsChain []CredentialsSource

var _ CredentialsSource = CredentialsChain{}

func (c CredentialsChain) Credentials(ctx Context, creds ...CredentialsSource) (Credentials, error) {
	if len(c) == 0 || reflect2.IsNil(c[0]) {
		return nil, nil
	}

	if len(creds) == 0 {
		return c[0].Credentials(ctx, c[1:]...)
	}
	return c[0].Credentials(ctx, append(append(c[:0:len(c)-1+len(creds)], c[1:]...), creds...))
}
