package manager

import (
	"context"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/gardener/landscaper/pkg/apis/config"
	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/registry/blueprints"
	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

// Interface describes the interface for a regapi manager.
// A regapi manager exposes itself a regapi api and delegates the
// request to the specific implementations.
type Interface interface {
	blueprintsregistry.Registry
	Set(schemaName string, scheme cdv2.TypedObjectCodec, registry blueprintsregistry.Registry) error
}

// New creates a new blueprint registry manager
func New(sharedCache cache.Cache) Interface {
	return &manager{
		registries:  map[string]blueprintsregistry.Registry{},
		sharedCache: sharedCache,
	}
}

// NewWithConfig creates a new regapi manager or the given regapi configuration
func NewWithConfig(log logr.Logger, config *config.RegistryConfiguration) (Interface, error) {
	m := &manager{
		registries: map[string]blueprintsregistry.Registry{},
	}

	if config.OCI.Cache != nil {
		sharedCache, err := cache.NewCache(log, cache.WithConfiguration(config.OCI.Cache))
		if err != nil {
			return nil, err
		}
		m.sharedCache = sharedCache
	}

	if config.OCI != nil {
		ociReg, err := blueprintsregistry.NewOCIRegistry(log, nil) // use the shared cache
		if err != nil {
			return nil, fmt.Errorf("unable to setup oci regapi: %w", err)
		}
		if err := m.Set(cdv2.OCIRegistryType, cdv2.KnownAccessTypes[cdv2.OCIRegistryType], ociReg); err != nil {
			return nil, err
		}
	}

	if config.Local != nil {
		local, err := blueprintsregistry.NewLocalRegistry(log, config.Local.ConfigPaths...)
		if err != nil {
			return nil, fmt.Errorf("unable to setup local regapi: %w", err)
		}
		if err := m.Set(blueprintsregistry.LocalAccessType, blueprintsregistry.LocalAccessCodec, local); err != nil {
			return nil, err
		}
	}

	return m, nil
}

type manager struct {
	registries  map[string]blueprintsregistry.Registry
	sharedCache cache.Cache
}

var _ Interface = &manager{}

func (m *manager) Set(schemaName string, scheme cdv2.TypedObjectCodec, registry blueprintsregistry.Registry) error {
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
		return nil, blueprintsregistry.NewWrongTypeError(ref.Access.GetType(), ref.Name, ref.Version, nil)
	}
	return reg.GetBlueprint(ctx, ref)
}

func (m *manager) GetContent(ctx context.Context, ref cdv2.Resource, fs vfs.FileSystem) error {
	reg, ok := m.registries[ref.Access.GetType()]
	if !ok {
		return blueprintsregistry.NewWrongTypeError(ref.Access.GetType(), ref.Name, ref.Version, nil)
	}
	return reg.GetContent(ctx, ref, fs)
}
