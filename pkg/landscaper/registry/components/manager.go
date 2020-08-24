// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package componentsregistry

import (
	"context"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

// Registry describes a component descriptor repository implementation
// that resolves component references.
type Registry interface {
	Resolve(ctx context.Context, repoCtx cdv2.RepositoryContext, ref cdv2.ObjectMeta) (*cdv2.ComponentDescriptor, error)
}

// TypedRegistry describes a repository ociClient that can handle the given type.
type TypedRegistry interface {
	Registry
	Type() string
}

type Manager struct {
	registries  map[string]Registry
	sharedCache cache.Cache
}

var _ Registry = &Manager{}

// New creates a ociClient that can handle multiple clients.
// The manager can handle a shared cache that is injected into the registries.
func New(sharedCache cache.Cache, clients ...TypedRegistry) (*Manager, error) {
	m := &Manager{}
	if err := m.Set(clients...); err != nil {
		return nil, err
	}
	return m, nil
}

// Set adds registries to the manager.
func (m *Manager) Set(registries ...TypedRegistry) error {
	for _, registry := range registries {
		if err := cache.InjectCacheInto(registry, m.sharedCache); err != nil {
			return err
		}
		m.registries[registry.Type()] = registry
	}
	return nil
}

// SharedCache returns the shared cache for all managed registries.
// Returns nil if there is no shared cache.
func (m *Manager) SharedCache() cache.Cache {
	return m.sharedCache
}

func (m *Manager) Resolve(ctx context.Context, repoCtx cdv2.RepositoryContext, ref cdv2.ObjectMeta) (*cdv2.ComponentDescriptor, error) {
	client, ok := m.registries[repoCtx.Type]
	if !ok {
		return nil, fmt.Errorf("unknown repository type %s", repoCtx.Type)
	}
	return client.Resolve(ctx, repoCtx, ref)
}
