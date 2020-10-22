// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package artifactsregistry

import (
	"context"
	"fmt"
	"io"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

// Registry is the interface for the landscaper to receive artifacts as blob data.
type Registry interface {
	// GetBlob returns the blob for a blob access method
	GetBlob(ctx context.Context, access cdv2.TypedObjectAccessor, writer io.Writer) (mediaType string, err error)
}

// TypedRegistry describes a registry that can handle the given access type.
type TypedRegistry interface {
	Registry
	Type() string
}

// Manager handles multiple artifact registries
type Manager struct {
	sharedCache cache.Cache
	registries  map[string]Registry
}

var _ Registry = &Manager{}

// New creates a new artifact blob registry manager
func New(sharedCache cache.Cache, registries ...TypedRegistry) (*Manager, error) {
	mgr := &Manager{
		sharedCache: sharedCache,
		registries:  map[string]Registry{},
	}
	if err := mgr.Add(registries...); err != nil {
		return nil, err
	}
	return mgr, nil
}

// Add an additional registry to the manager.
func (m *Manager) Add(registries ...TypedRegistry) error {
	for _, reg := range registries {
		if err := cache.InjectCacheInto(reg, m.sharedCache); err != nil {
			return fmt.Errorf("unable to add registry for '%s': %w", reg.Type(), err)
		}
		m.registries[reg.Type()] = reg
	}
	return nil
}

func (m *Manager) GetBlob(ctx context.Context, access cdv2.TypedObjectAccessor, writer io.Writer) (string, error) {
	reg, ok := m.registries[access.GetType()]
	if !ok {
		return "", fmt.Errorf("no registry registered that can handle the access of type %s", access.GetType())
	}
	return reg.GetBlob(ctx, access, writer)
}
