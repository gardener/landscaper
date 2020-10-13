// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package blueprintsregistry

import (
	"context"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/gardener/landscaper/pkg/apis/config"
	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

// Manager describes the interface for a blueprints registry manager.
// A blueprints registry manager exposes itself a blueprints registry api and delegates the
// request to the specific implementations.
type Manager interface {
	Registry
	Set(schemaName string, scheme cdv2.TypedObjectCodec, registry Registry) error
}

// New creates a new blueprint registry manager
func New(sharedCache cache.Cache) Manager {
	return &manager{
		registries:  map[string]Registry{},
		sharedCache: sharedCache,
	}
}

// NewWithConfig creates a new regapi manager or the given regapi configuration
func NewWithConfig(log logr.Logger, config *config.RegistryConfiguration) (Manager, error) {
	m := &manager{
		registries: map[string]Registry{},
	}

	if config.OCI.Cache != nil {
		sharedCache, err := cache.NewCache(log, cache.WithConfiguration(config.OCI.Cache))
		if err != nil {
			return nil, err
		}
		m.sharedCache = sharedCache
	}

	if config.OCI != nil {
		ociReg, err := NewOCIRegistry(log, nil) // use the shared cache
		if err != nil {
			return nil, fmt.Errorf("unable to setup oci regapi: %w", err)
		}
		if err := m.Set(cdv2.OCIRegistryType, cdv2.KnownAccessTypes[cdv2.OCIRegistryType], ociReg); err != nil {
			return nil, err
		}
	}

	if config.Local != nil {
		local, err := NewLocalRegistry(log, config.Local.ConfigPaths...)
		if err != nil {
			return nil, fmt.Errorf("unable to setup local regapi: %w", err)
		}
		if err := m.Set(LocalAccessType, LocalAccessCodec, local); err != nil {
			return nil, err
		}
	}

	return m, nil
}

type manager struct {
	registries  map[string]Registry
	sharedCache cache.Cache
}

var _ Manager = &manager{}

func (m *manager) Set(schemaName string, scheme cdv2.TypedObjectCodec, registry Registry) error {
	cdv2.KnownAccessTypes[schemaName] = scheme
	m.registries[schemaName] = registry
	return cache.InjectCacheInto(registry, m.sharedCache)
}

// SharedCache returns the shared cache for all managed registries.
// Returns nil if there is no shared cache.
func (m *manager) SharedCache() cache.Cache {
	return m.sharedCache
}

func (m *manager) GetBlueprint(ctx context.Context, ref cdv2.Resource) (*v1alpha1.Blueprint, error) {
	reg, ok := m.registries[ref.Access.GetType()]
	if !ok {
		return nil, NewWrongTypeError(ref.Access.GetType(), ref.Name, ref.Version, nil)
	}
	return reg.GetBlueprint(ctx, ref)
}

func (m *manager) GetContent(ctx context.Context, ref cdv2.Resource, fs vfs.FileSystem) error {
	reg, ok := m.registries[ref.Access.GetType()]
	if !ok {
		return NewWrongTypeError(ref.Access.GetType(), ref.Name, ref.Version, nil)
	}
	return reg.GetContent(ctx, ref, fs)
}
