// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package memory

import (
	"sync"

	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
)

type Repository struct {
	lock        sync.RWMutex
	name        string
	credentials map[string]cpi.Credentials
}

func NewRepository(name string) *Repository {
	return &Repository{
		name:        name,
		credentials: map[string]cpi.Credentials{},
	}
}

var _ cpi.Repository = &Repository{}

func (r *Repository) ExistsCredentials(name string) (bool, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	_, ok := r.credentials[name]
	return ok, nil
}

func (r *Repository) LookupCredentials(name string) (cpi.Credentials, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	c, ok := r.credentials[name]
	if ok {
		return cpi.NewCredentials(c.Properties()), nil
	}
	return nil, cpi.ErrUnknownCredentials(name)
}

func (r *Repository) WriteCredentials(name string, creds cpi.Credentials) (cpi.Credentials, error) {
	c := cpi.NewCredentials(creds.Properties())
	r.lock.Lock()
	defer r.lock.Unlock()
	r.credentials[name] = c
	return c, nil
}
