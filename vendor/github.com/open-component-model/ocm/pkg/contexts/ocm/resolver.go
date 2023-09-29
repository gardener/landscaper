// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"sync"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/internal"
	"github.com/open-component-model/ocm/pkg/errors"
)

type CompoundResolver struct {
	lock      sync.RWMutex
	resolvers []ComponentVersionResolver
}

var _ ComponentVersionResolver = (*CompoundResolver)(nil)

func NewCompoundResolver(res ...ComponentVersionResolver) ComponentVersionResolver {
	for i := 0; i < len(res); i++ {
		if res[i] == nil {
			res = append(res[:i], res[i+1:]...)
			i--
		}
	}
	if len(res) == 1 {
		return res[0]
	}
	return &CompoundResolver{resolvers: res}
}

func (c *CompoundResolver) LookupComponentVersion(name string, version string) (internal.ComponentVersionAccess, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	for _, r := range c.resolvers {
		if r == nil {
			continue
		}
		cv, err := r.LookupComponentVersion(name, version)
		if err == nil && cv != nil {
			return cv, nil
		}
		if !errors.IsErrNotFoundKind(err, KIND_COMPONENTVERSION) {
			return nil, err
		}
	}
	return nil, errors.ErrNotFound(KIND_OCM_REFERENCE, common.NewNameVersion(name, version).String())
}

func (c *CompoundResolver) AddResolver(r ComponentVersionResolver) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.resolvers = append(c.resolvers, r)
}

////////////////////////////////////////////////////////////////////////////////

type MatchingResolver interface {
	ComponentVersionResolver
	ContextProvider

	AddRule(prefix string, spec RepositorySpec, prio ...int)
	Finalize() error
}

func NewMatchingResolver(ctx ContextProvider) MatchingResolver {
	return internal.NewMatchingResolver(ctx.OCMContext())
}
