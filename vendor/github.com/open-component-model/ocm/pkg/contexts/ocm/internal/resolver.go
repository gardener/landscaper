// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"strings"
	"sync"

	"golang.org/x/exp/slices"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/finalizer"
	"github.com/open-component-model/ocm/pkg/registrations"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

////////////////////////////////////////////////////////////////////////////////

type ResolverRule struct {
	prefix string
	path   registrations.NamePath
	spec   RepositorySpec
	prio   int
}

func (r *ResolverRule) GetPrefix() string {
	return r.prefix
}

func (r *ResolverRule) GetSpecification() RepositorySpec {
	return r.spec
}

func (r *ResolverRule) GetPriority() int {
	return r.prio
}

type RepositoryCache struct {
	lock         sync.Mutex
	finalize     finalizer.Finalizer
	repositories map[datacontext.ObjectKey]Repository
}

func NewRepositoryCache() *RepositoryCache {
	return &RepositoryCache{
		repositories: map[datacontext.ObjectKey]Repository{},
	}
}

func (c *RepositoryCache) LookupRepository(ctx Context, spec RepositorySpec) (Repository, error) {
	spec, err := ctx.RepositoryTypes().Convert(spec)
	if err != nil {
		return nil, err
	}
	data, err := runtime.DefaultJSONEncoding.Marshal(spec)
	if err != nil {
		return nil, err
	}
	key := datacontext.ObjectKey{
		Object: ctx,
		Name:   string(data),
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	if r := c.repositories[key]; r != nil {
		return r, nil
	}
	repo, err := ctx.RepositoryForSpec(spec)
	if err != nil {
		return nil, err
	}
	c.repositories[key] = repo
	c.finalize.Close(repo)
	return repo, err
}

func (c *RepositoryCache) Finalize() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	err := c.finalize.Finalize()
	c.repositories = map[datacontext.ObjectKey]Repository{}
	return err
}

func NewResolverRule(prefix string, spec RepositorySpec, prio ...int) *ResolverRule {
	p := registrations.NewNamePath(prefix)
	return &ResolverRule{
		prefix: prefix,
		path:   p,
		spec:   spec,
		prio:   utils.OptionalDefaulted(10, prio...),
	}
}

func (r *ResolverRule) Compare(o *ResolverRule) int {
	if d := r.prio - o.prio; d != 0 {
		return d
	}
	return r.path.Compare(o.path)
}

func (r *ResolverRule) Match(name string) bool {
	return r.prefix == "" || r.prefix == name || strings.HasPrefix(name, r.prefix+"/")
}

type MatchingResolver struct {
	lock  sync.Mutex
	ctx   Context
	cache *RepositoryCache
	rules []*ResolverRule
}

func NewMatchingResolver(ctx ContextProvider, rules ...*ResolverRule) *MatchingResolver {
	return &MatchingResolver{
		lock:  sync.Mutex{},
		ctx:   ctx.OCMContext(),
		cache: NewRepositoryCache(),
		rules: nil,
	}
}

func (r *MatchingResolver) OCMContext() Context {
	return r.ctx
}

func (r *MatchingResolver) Finalize() error {
	return r.cache.Finalize()
}

func (r *MatchingResolver) GetRules() []*ResolverRule {
	r.lock.Lock()
	defer r.lock.Unlock()
	return slices.Clone(r.rules)
}

func (r *MatchingResolver) AddRule(prefix string, spec RepositorySpec, prio ...int) {
	r.lock.Lock()
	defer r.lock.Unlock()

	rule := NewResolverRule(prefix, spec, prio...)
	found := len(r.rules)
	for i, o := range r.rules {
		if o.Compare(rule) < 0 {
			found = i
			break
		}
	}
	r.rules = slices.Insert(r.rules, found, rule)
}

func (r *MatchingResolver) LookupComponentVersion(name string, version string) (ComponentVersionAccess, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for _, rule := range r.rules {
		if rule.Match(name) {
			repo, err := r.cache.LookupRepository(r.ctx, rule.spec)
			if err != nil {
				return nil, err
			}
			cv, err := repo.LookupComponentVersion(name, version)
			if err == nil && cv != nil {
				return cv, nil
			}
			if !errors.IsErrNotFoundKind(err, KIND_COMPONENTVERSION) {
				return nil, err
			}
		}
	}
	return nil, errors.ErrNotFound(KIND_COMPONENTVERSION, common.NewNameVersion(name, version).String())
}
