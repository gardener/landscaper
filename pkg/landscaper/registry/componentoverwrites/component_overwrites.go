// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentoverwrites

import (
	"sync"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// Overwriter is a interface that implements a component reference replace method.
type Overwriter interface {
	Replace(reference *lsv1alpha1.ComponentDescriptorReference) (bool, error)
}

// Manager is a manager that manages all component overwrites.
type Manager struct {
	mux        sync.RWMutex
	overwrites map[string]lsv1alpha1.ComponentOverwrite
}

// New creates a new component overwrite manager.
func New() *Manager {
	return &Manager{
		overwrites: map[string]lsv1alpha1.ComponentOverwrite{},
	}
}

// Replace replaces a component version and target if defined.
func (m *Manager) Replace(reference *lsv1alpha1.ComponentDescriptorReference) (bool, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	entry, ok := m.overwrites[reference.ComponentName]
	if !ok {
		return false, nil
	}

	if len(entry.Component.Version) != 0 && entry.Component.Version != reference.Version {
		return false, nil
	}
	if entry.Component.RepositoryContext != nil && *entry.Component.RepositoryContext != *reference.RepositoryContext {
		return false, nil
	}
	if entry.Target.RepositoryContext != nil {
		reference.RepositoryContext = entry.Target.RepositoryContext
	}
	if len(entry.Target.ComponentName) != 0 {
		reference.ComponentName = entry.Target.ComponentName
	}
	if len(entry.Target.Version) != 0 {
		reference.Version = entry.Target.Version
	}
	return true, nil
}

// Add adds or updates a component overwrite.
func (m *Manager) Add(overwrite lsv1alpha1.ComponentOverwrite) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.overwrites[overwrite.Component.ComponentName] = overwrite
}
