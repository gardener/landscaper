// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"sync"
)

type ParseHandler func(u *UniformRepositorySpec) error

type registry struct {
	lock     sync.RWMutex
	handlers map[string]ParseHandler
}

func (r *registry) Register(ty string, h ParseHandler) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.handlers[ty] = h
}

func (r *registry) Get(ty string) ParseHandler {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.handlers[ty]
}

func (r *registry) Handle(u UniformRepositorySpec) (UniformRepositorySpec, error) {
	h := r.Get(u.Type)
	if h != nil {
		err := h(&u)
		return u, err
	}
	return u, nil
}

var parseregistry = &registry{handlers: map[string]ParseHandler{}}

func RegisterRefParseHandler(ty string, h ParseHandler) {
	parseregistry.Register(ty, h)
}

func GetRefParseHandler(ty string, h ParseHandler) {
	parseregistry.Get(ty)
}

func HandleRef(u UniformRepositorySpec) (UniformRepositorySpec, error) {
	return parseregistry.Handle(u)
}
