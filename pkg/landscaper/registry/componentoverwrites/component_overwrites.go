// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentoverwrites

import (
	"fmt"
	"sort"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// OverwriterFunc describes a simple func that implements the overwriter interface.
type OverwriterFunc func(reference *lsv1alpha1.ComponentDescriptorReference) bool

func (f OverwriterFunc) Replace(reference *lsv1alpha1.ComponentDescriptorReference) bool {
	return f(reference)
}

// Manager is a manager that manages all component overwrites.
type Manager struct {
	mux        sync.RWMutex
	overwrites map[string]*EvaluatedOverwrites
	effective  *Substitutions
}

type EvaluatedOverwrites struct {
	timestamp     metav1.Time
	substitutions *Substitutions
}

func NewClusterSubstitutions(subs []lsv1alpha1.ComponentOverwrite) *Substitutions {
	newSubs := make([]lsv1alpha1.ComponentVersionOverwrite, len(subs))
	for i, s := range subs {
		newSubs[i] = lsv1alpha1.ComponentVersionOverwrite{
			Source: lsv1alpha1.ComponentVersionOverwriteReference{
				RepositoryContext: s.Component.RepositoryContext,
				ComponentName:     s.Component.ComponentName,
				Version:           s.Component.Version,
			},
			Substitution: lsv1alpha1.ComponentVersionOverwriteReference{
				RepositoryContext: s.Target.RepositoryContext,
				ComponentName:     s.Target.ComponentName,
				Version:           s.Target.Version,
			},
		}
	}
	return NewSubstitutions(newSubs)
}

// New creates a new component overwrite manager.
func New() *Manager {
	return &Manager{
		overwrites: map[string]*EvaluatedOverwrites{},
		effective:  NewSubstitutions(nil),
	}
}

// Replace replaces a component version and target if defined.
func (m *Manager) GetOverwriter() Overwriter {
	m.mux.RLock()
	defer m.mux.RUnlock()
	return m.effective
}

// Add adds or updates a component overwrite.
func (m *Manager) Add(overwrites *lsv1alpha1.ComponentOverwrites) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.overwrites[overwrites.Name] = &EvaluatedOverwrites{
		timestamp:     overwrites.CreationTimestamp,
		substitutions: NewClusterSubstitutions(overwrites.Overwrites),
	}
	m.merge()
}

func (m *Manager) Delete(name string) {
	m.mux.Lock()
	defer m.mux.Unlock()
	delete(m.overwrites, name)
	m.merge()
}

func (m *Manager) merge() {
	sorted := make([]*EvaluatedOverwrites, len(m.overwrites))
	i := 0
	for _, v := range m.overwrites {
		sorted[i] = v
		i++
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[j].timestamp.Before(&sorted[i].timestamp)
	})
	m.effective = NewSubstitutions(nil)
	for _, elem := range sorted {
		m.effective.Substitutions = append(m.effective.Substitutions, elem.substitutions.Substitutions...)
	}
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
