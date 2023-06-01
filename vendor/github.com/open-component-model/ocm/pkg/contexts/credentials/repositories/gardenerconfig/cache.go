// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package gardenerconfig

import (
	"fmt"
	"sync"

	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	gardenercfgcpi "github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/gardenerconfig/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
)

const ATTR_REPOS = "github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/gardenerconfig"

type Repositories struct {
	lock  sync.Mutex
	repos map[string]*Repository
}

func newRepositories(datacontext.Context) interface{} {
	return &Repositories{
		repos: map[string]*Repository{},
	}
}

func (r *Repositories) GetRepository(ctx cpi.Context, url string, configType gardenercfgcpi.ConfigType, cipher Cipher, key []byte, propagateConsumerIdentity bool) (*Repository, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if _, ok := r.repos[url]; !ok {
		repo, err := NewRepository(ctx, url, configType, cipher, key, propagateConsumerIdentity)
		if err != nil {
			return nil, fmt.Errorf("unable to create repository: %w", err)
		}
		r.repos[url] = repo
	}
	return r.repos[url], nil
}
