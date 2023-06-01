// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package memory

import (
	"sync"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
)

const ATTR_REPOS = "github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/memory"

type Repositories struct {
	lock  sync.Mutex
	repos map[string]*Repository
}

func newRepositories(datacontext.Context) interface{} {
	return &Repositories{
		repos: map[string]*Repository{},
	}
}

func (r *Repositories) GetRepository(name string) *Repository {
	r.lock.Lock()
	defer r.lock.Unlock()
	repo := r.repos[name]
	if repo == nil {
		repo = NewRepository(name)
		r.repos[name] = repo
	}
	return repo
}
