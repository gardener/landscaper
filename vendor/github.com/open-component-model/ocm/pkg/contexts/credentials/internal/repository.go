// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/generics"
)

type Repository interface {
	ExistsCredentials(name string) (bool, error)
	LookupCredentials(name string) (Credentials, error)
	WriteCredentials(name string, creds Credentials) (Credentials, error)
}

type Credentials interface {
	CredentialsSource
	ExistsProperty(name string) bool
	GetProperty(name string) string
	PropertyNames() generics.Set[string]
	Properties() common.Properties
}

type DirectCredentials common.Properties

var _ Credentials = (*DirectCredentials)(nil)

func NewCredentials(props common.Properties) DirectCredentials {
	if props == nil {
		props = common.Properties{}
	} else {
		props = props.Copy()
	}
	return DirectCredentials(props)
}

func (c DirectCredentials) ExistsProperty(name string) bool {
	_, ok := c[name]
	return ok
}

func (c DirectCredentials) GetProperty(name string) string {
	return c[name]
}

func (c DirectCredentials) PropertyNames() generics.Set[string] {
	return common.Properties(c).Names()
}

func (c DirectCredentials) Properties() common.Properties {
	return common.Properties(c).Copy()
}

func (c DirectCredentials) Credentials(Context, ...CredentialsSource) (Credentials, error) {
	return c, nil
}

func (c DirectCredentials) Copy() DirectCredentials {
	return DirectCredentials(common.Properties(c).Copy())
}

func (c DirectCredentials) String() string {
	return common.Properties(c).String()
}
