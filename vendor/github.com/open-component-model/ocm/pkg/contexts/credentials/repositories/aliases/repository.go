// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package aliases

import (
	"sync"

	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
)

type Repository struct {
	sync.Mutex
	name  string
	spec  cpi.RepositorySpec
	creds cpi.CredentialsSource
	repo  cpi.Repository
}

func (a *Repository) GetRepository(ctx cpi.Context, creds cpi.Credentials) (cpi.Repository, error) {
	a.Lock()
	defer a.Unlock()
	if a.repo != nil {
		return a.repo, nil
	}

	src := cpi.CredentialsChain{}
	if a.creds != nil {
		src = append(src, a.creds)
	}
	if creds != nil {
		src = append(src, creds)
	}
	repo, err := ctx.RepositoryForSpec(a.spec, src...)
	if err != nil {
		return nil, err
	}
	a.repo = repo
	return repo, nil
}

func NewRepository(name string, spec cpi.RepositorySpec, creds cpi.Credentials) *Repository {
	return &Repository{
		name:  name,
		spec:  spec,
		creds: creds,
	}
}
