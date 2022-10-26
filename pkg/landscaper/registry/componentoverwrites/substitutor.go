// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentoverwrites

import (
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// Overwriter is a interface that implements a component reference replace method.
type Overwriter interface {
	Replace(reference *lsv1alpha1.ComponentDescriptorReference) bool
}

// Substitutions handles overwrites and implements the Substitutor interface.
type Substitutions struct {
	Substitutions []lsv1alpha1.ComponentVersionOverwrite
}

func NewSubstitutions(subs []lsv1alpha1.ComponentVersionOverwrite) *Substitutions {
	return &Substitutions{
		Substitutions: subs,
	}
}

func matches(matchRef *lsv1alpha1.ComponentVersionOverwriteReference, obj *lsv1alpha1.ComponentDescriptorReference) bool {
	if len(matchRef.ComponentName) != 0 && matchRef.ComponentName != obj.ComponentName {
		return false
	}
	if len(matchRef.Version) != 0 && matchRef.Version != obj.Version {
		return false
	}
	if matchRef.RepositoryContext != nil && !cdv2.UnstructuredTypesEqual(matchRef.RepositoryContext, obj.RepositoryContext) {
		return false
	}
	return true
}

func mergeCDReference(mergeRef *lsv1alpha1.ComponentVersionOverwriteReference, obj *lsv1alpha1.ComponentDescriptorReference) {
	// don't merge any field if we cannot merge all which are provided
	if (len(mergeRef.ComponentName) != 0 && len(obj.ComponentName) != 0) ||
		(len(mergeRef.Version) != 0 && len(obj.Version) != 0) ||
		(mergeRef.RepositoryContext != nil && obj.RepositoryContext != nil) {
		return
	}

	// since we know that there are no conflicts, we can just merge the given fields
	if len(mergeRef.ComponentName) != 0 {
		obj.ComponentName = mergeRef.ComponentName
	}
	if len(mergeRef.Version) != 0 {
		obj.Version = mergeRef.Version
	}
	if mergeRef.RepositoryContext != nil {
		obj.RepositoryContext = mergeRef.RepositoryContext
	}
}

func (sm *Substitutions) Replace(ref *lsv1alpha1.ComponentDescriptorReference) bool {
	merge := &lsv1alpha1.ComponentDescriptorReference{}
	changed := false
	for _, subs := range sm.Substitutions {
		if matches(&subs.Source, ref) {
			changed = true
			mergeCDReference(&subs.Substitution, merge)
			if merge.RepositoryContext != nil && len(merge.ComponentName) != 0 && len(merge.Version) != 0 {
				break
			}
		}
	}
	if !changed {
		return false
	}

	if len(merge.ComponentName) != 0 {
		ref.ComponentName = merge.ComponentName
	}
	if len(merge.Version) != 0 {
		ref.Version = merge.Version
	}
	if merge.RepositoryContext != nil {
		ref.RepositoryContext = merge.RepositoryContext
	}

	return true
}
