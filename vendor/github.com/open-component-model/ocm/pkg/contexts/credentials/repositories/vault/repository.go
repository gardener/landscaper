// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package vault

import (
	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/internal"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/vault/identity"
	"github.com/open-component-model/ocm/pkg/errors"
)

type Repository struct {
	ctx      cpi.Context
	spec     *RepositorySpec
	id       cpi.ConsumerIdentity
	provider *ConsumerProvider
}

var (
	_ cpi.Repository               = (*Repository)(nil)
	_ cpi.ConsumerIdentityProvider = (*Repository)(nil)
)

func NewRepository(ctx cpi.Context, spec *RepositorySpec) (*Repository, error) {
	id, err := identity.GetConsumerId(spec.ServerURL, spec.Namespace, spec.SecretsEngine, spec.Path)
	if err != nil {
		return nil, err
	}
	r := &Repository{
		ctx:  ctx,
		spec: spec,
		id:   id,
	}
	if spec.ServerURL == "" {
		return nil, errors.ErrInvalid("server url")
	}
	r.provider, err = NewConsumerProvider(r)
	if err != nil {
		return nil, err
	}
	if spec.PropgateConsumerIdentity {
		ctx.RegisterConsumerProvider(spec.GetKey(), r.provider)
	}
	return r, err
}

var _ cpi.Repository = &Repository{}

func (r *Repository) ExistsCredentials(name string) (bool, error) {
	return r.provider.ExistsCredentials(name)
}

func (r *Repository) LookupCredentials(name string) (cpi.Credentials, error) {
	return r.provider.LookupCredentials(name)
}

func (r *Repository) WriteCredentials(name string, creds cpi.Credentials) (cpi.Credentials, error) {
	return nil, errors.ErrNotSupported("write", "credentials", Type)
}

func (r *Repository) GetConsumerId(uctx ...internal.UsageContext) internal.ConsumerIdentity {
	return r.id
}

func (r *Repository) GetIdentityMatcher() string {
	return identity.CONSUMER_TYPE
}
