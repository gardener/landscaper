// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

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
	if m.registries == nil {
		m.registries = map[string]Registry{}
	}
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
