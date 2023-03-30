// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package aliases

import (
	"sync"

	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
)

const ATTR_REPOS = "github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/aliases"

type Repositories struct {
	sync.RWMutex
	repos map[string]*Repository
}

func newRepositories(datacontext.Context) interface{} {
	return &Repositories{
		repos: map[string]*Repository{},
	}
}

func (c *Repositories) GetRepository(name string) *Repository {
	c.RLock()
	defer c.RUnlock()
	return c.repos[name]
}

func (c *Repositories) Set(name string, spec cpi.RepositorySpec, creds cpi.CredentialsSource) {
	c.Lock()
	defer c.Unlock()
	c.repos[name] = &Repository{
		name:  name,
		spec:  spec,
		creds: creds,
	}
}
