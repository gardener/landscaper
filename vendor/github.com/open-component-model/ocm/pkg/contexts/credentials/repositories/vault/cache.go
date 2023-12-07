// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package vault

import (
	"sync"

	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/errors"
)

const ATTR_REPOS = "github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/vault"

type Repositories struct {
	lock  sync.Mutex
	repos map[cpi.ProviderIdentity]*Repository
}

func newRepositories(datacontext.Context) interface{} {
	return &Repositories{
		repos: map[cpi.ProviderIdentity]*Repository{},
	}
}

func (r *Repositories) GetRepository(ctx cpi.Context, spec *RepositorySpec) (*Repository, error) {
	var repo *Repository

	if spec.ServerURL == "" {
		return nil, errors.ErrInvalid("server url")
	}
	r.lock.Lock()
	defer r.lock.Unlock()

	var err error
	key := spec.GetKey()
	repo = r.repos[key]
	if repo == nil {
		repo, err = NewRepository(ctx, spec)
		if err == nil {
			r.repos[key] = repo
		}
	}
	return repo, err
}
