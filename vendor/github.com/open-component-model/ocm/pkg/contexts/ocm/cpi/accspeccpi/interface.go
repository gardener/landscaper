// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accspeccpi

import (
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/internal"
)

type (
	Context         = internal.Context
	ContextProvider = internal.ContextProvider

	AccessType = internal.AccessType

	AccessMethodImpl = internal.AccessMethodImpl
	AccessMethod     = internal.AccessMethod
	AccessSpec       = internal.AccessSpec
	AccessSpecRef    = internal.AccessSpecRef

	HintProvider            = internal.HintProvider
	GlobalAccessProvider    = internal.GlobalAccessProvider
	CosumerIdentityProvider = credentials.ConsumerIdentityProvider

	ComponentVersionAccess = internal.ComponentVersionAccess
)

var (
	newStrictAccessTypeScheme = internal.NewStrictAccessTypeScheme
	defaultAccessTypeScheme   = internal.DefaultAccessTypeScheme
)

func NewAccessSpecRef(spec AccessSpec) *AccessSpecRef {
	return internal.NewAccessSpecRef(spec)
}
