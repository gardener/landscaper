package componentresolvers

import (
	"context"
	"fmt"

	"github.com/gardener/component-cli/ociclient/cache"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
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

// SetRegistryForAlias enables using a specific registry implementation for a different type than its statically
// configured type (the type returned by registry.Type() )
func (m *Manager) SetRegistryForAlias(registry TypedRegistry, alias string) error {
	if m.registries == nil {
		m.registries = map[string]ctf.ComponentResolver{}
	}
	m.registries[alias] = registry
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
