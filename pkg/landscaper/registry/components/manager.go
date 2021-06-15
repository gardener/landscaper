// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentsregistry

import (
	"context"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"

	"github.com/gardener/component-cli/ociclient/cache"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/pkg/utils"
)

// TypedRegistry describes a registry that can handle the given type.
type TypedRegistry interface {
	ctf.ComponentResolver
	Type() string
}

type Manager struct {
	registries  map[string]ctf.ComponentResolver
	sharedCache cache.Cache
}

var _ ctf.ComponentResolver = &Manager{}

// New creates a ociClient that can handle multiple clients.
// The manager can handle a shared cache that is injected into the registries.
func New(sharedCache cache.Cache, clients ...TypedRegistry) (*Manager, error) {
	m := &Manager{
		sharedCache: sharedCache,
	}
	if err := m.Set(clients...); err != nil {
		return nil, err
	}
	return m, nil
}

// Set adds registries to the manager.
func (m *Manager) Set(registries ...TypedRegistry) error {
	if m.registries == nil {
		m.registries = map[string]ctf.ComponentResolver{}
	}
	for _, registry := range registries {
		m.registries[registry.Type()] = registry
	}
	return nil
}

// SharedCache returns the shared cache for all managed registries.
// Returns nil if there is no shared cache.
func (m *Manager) SharedCache() cache.Cache {
	return m.sharedCache
}

func (m *Manager) Resolve(ctx context.Context, repoCtx cdv2.Repository, name, version string) (*cdv2.ComponentDescriptor, error) {
	client, ok := m.registries[repoCtx.GetType()]
	if !ok {
		return nil, fmt.Errorf("unknown repository type %s", repoCtx.GetType())
	}
	return client.Resolve(ctx, repoCtx, name, version)
}

func (m *Manager) ResolveWithBlobResolver(ctx context.Context, repoCtx cdv2.Repository, name, version string) (*cdv2.ComponentDescriptor, ctf.BlobResolver, error) {
	client, ok := m.registries[repoCtx.GetType()]
	if !ok {
		return nil, nil, fmt.Errorf("unknown repository type %s", repoCtx.GetType())
	}
	return client.ResolveWithBlobResolver(ctx, repoCtx, name, version)
}

// SetupManagerFromConfig returns a new Manager instance initialized with the given OCI configuration
func SetupManagerFromConfig(log logr.Logger, config *config.OCIConfiguration, cacheIdentifier string) (*Manager, error) {
	var sharedCache cache.Cache
	if config != nil && config.Cache != nil {
		var err error
		sharedCache, err = cache.NewCache(log, utils.ToOCICacheOptions(config.Cache, cacheIdentifier)...)
		if err != nil {
			return nil, err
		}
	}
	return New(sharedCache)
}
