// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentoverwrites

import (
	"fmt"
	"sync"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// Overwriter is a interface that implements a component reference replace method.
type Overwriter interface {
	Replace(reference *lsv1alpha1.ComponentDescriptorReference) (bool, error)
}

// OverwriterFunc describes a simple func that implements the overwriter interface.
type OverwriterFunc func(reference *lsv1alpha1.ComponentDescriptorReference) (bool, error)

func (f OverwriterFunc) Replace(reference *lsv1alpha1.ComponentDescriptorReference) (bool, error) {
	return f(reference)
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
	if entry.Component.RepositoryContext != nil && !cdv2.UnstructuredTypesEqual(entry.Component.RepositoryContext, reference.RepositoryContext) {
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

// ReferenceDiff returns a human readable diff for two refs
func ReferenceDiff(oldRef, newRef *lsv1alpha1.ComponentDescriptorReference) string {
	var (
		repoCtx       = "RepositoryContext not updated"
		componentName = "ComponentName not updated"
		version       = "Version not updated"
	)

	if !cdv2.UnstructuredTypesEqual(oldRef.RepositoryContext, newRef.RepositoryContext) {
		repoCtx = repositoryContextDiff(oldRef.RepositoryContext, newRef.RepositoryContext)
	}
	if oldRef.ComponentName != newRef.ComponentName {
		componentName = fmt.Sprintf("%s -> %s", oldRef.ComponentName, newRef.ComponentName)
	}
	if oldRef.Version != newRef.Version {
		componentName = fmt.Sprintf("%s -> %s", oldRef.Version, newRef.Version)
	}
	return fmt.Sprintf(`
Componenten reference has been overwritten:
%s
%s
%s
`, repoCtx, componentName, version)
}

func repositoryContextDiff(oldCtx, newCtx *cdv2.UnstructuredTypedObject) string {
	if oldCtx == nil {
		return "-> " + string(newCtx.Raw)
	}
	defaultDiff := fmt.Sprintf("%s -> %s", string(oldCtx.Raw), string(newCtx.Raw))
	// use specific behavior if both repository contexts are ociregistries
	if oldCtx.GetType() == cdv2.OCIRegistryType &&
		newCtx.GetType() == cdv2.OCIRegistryType {
		oldOciReg := cdv2.OCIRegistryRepository{}
		if err := oldCtx.DecodeInto(&oldOciReg); err != nil {
			return defaultDiff
		}
		newOciReg := cdv2.OCIRegistryRepository{}
		if err := newCtx.DecodeInto(&newOciReg); err != nil {
			return defaultDiff
		}
		return fmt.Sprintf("%s (%s) -> %s (%s)", oldOciReg.BaseURL, oldOciReg.ComponentNameMapping, newOciReg.BaseURL, newOciReg.ComponentNameMapping)
	}
	return defaultDiff
}
