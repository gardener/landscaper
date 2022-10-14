// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentoverwrites

import (
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// OverwriterFunc describes a simple func that implements the overwriter interface.
type OverwriterFunc func(reference *lsv1alpha1.ComponentDescriptorReference) bool

func (f OverwriterFunc) Replace(reference *lsv1alpha1.ComponentDescriptorReference) bool {
	return f(reference)
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
