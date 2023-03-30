// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dockerconfig

import (
	dockercred "github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/generics"
)

type Credentials struct {
	repo  *Repository
	name  string
	store dockercred.Store
}

var _ cpi.Credentials = (*Credentials)(nil)

// NewCredentials describes a default getter method for a authentication method.
func NewCredentials(repo *Repository, name string, store dockercred.Store) cpi.Credentials {
	return &Credentials{
		repo:  repo,
		name:  name,
		store: store,
	}
}

func (c *Credentials) get() common.Properties {
	auth, err := c.repo.config.GetAuthConfig(c.name)
	if err != nil {
		return common.Properties{}
	}
	return newCredentials(auth).Properties()
}

func (c *Credentials) Credentials(context cpi.Context, source ...cpi.CredentialsSource) (cpi.Credentials, error) {
	var auth types.AuthConfig
	var err error
	if c.store == nil {
		auth, err = c.repo.config.GetAuthConfig(c.name)
	} else {
		auth, err = c.store.Get(c.name)
	}
	if err != nil {
		return nil, err
	}
	return newCredentials(auth), nil
}

func (c *Credentials) ExistsProperty(name string) bool {
	_, ok := c.get()[name]
	return ok
}

func (c *Credentials) GetProperty(name string) string {
	return c.get()[name]
}

func (c *Credentials) PropertyNames() generics.Set[string] {
	return c.get().Names()
}

func (c *Credentials) Properties() common.Properties {
	return c.get()
}
