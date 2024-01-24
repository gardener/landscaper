// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package composition

import (
	"sync"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/finalizer"
	"github.com/open-component-model/ocm/pkg/refmgmt"
)

const ATTR_REPOS = "github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/composition"

type Repositories struct {
	lock  sync.Mutex
	repos map[string]cpi.Repository
}

var _ finalizer.Finalizable = (*Repositories)(nil)

func newRepositories(datacontext.Context) interface{} {
	return &Repositories{
		repos: map[string]cpi.Repository{},
	}
}

func (r *Repositories) GetRepository(name string) cpi.Repository {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.repos[name]
}

func (r *Repositories) SetRepository(name string, repo cpi.Repository) {
	r.lock.Lock()
	defer r.lock.Unlock()

	old := r.repos[name]
	if old != nil {
		refmgmt.AsLazy(old).Close()
	}
	r.repos[name] = repo
}

func (r *Repositories) Finalize() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	list := errors.ErrListf("composition repositories")
	for n, r := range r.repos {
		list.Addf(nil, refmgmt.AsLazy(r).Close(), "repository %s", n)
	}

	r.repos = map[string]cpi.Repository{}
	return list.Result()
}

func Cleanup(ctx cpi.ContextProvider) error {
	repos := ctx.OCMContext().GetAttributes().GetAttribute(ATTR_REPOS)
	if repos != nil {
		return repos.(*Repositories).Finalize()
	}
	return nil
}
