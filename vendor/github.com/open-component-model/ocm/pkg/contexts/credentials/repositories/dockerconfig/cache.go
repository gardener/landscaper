// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dockerconfig

import (
	"sync"

	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
)

const ATTR_REPOS = "github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/dockerconfig"

type Repositories struct {
	lock  sync.Mutex
	repos map[string]*Repository
}

func newRepositories(datacontext.Context) interface{} {
	return &Repositories{
		repos: map[string]*Repository{},
	}
}

func (r *Repositories) GetRepository(ctx cpi.Context, name string, data []byte, propagate bool) (*Repository, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	var (
		err  error = nil
		repo *Repository
	)
	if name != "" {
		repo = r.repos[name]
	}
	if repo == nil {
		repo, err = NewRepository(ctx, name, data, propagate)
		if err == nil {
			r.repos[name] = repo
		}
	}
	return repo, err
}
