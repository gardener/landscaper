// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package targetresolver

import (
	"context"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

type TargetResolver interface {
	// Resolve resolves the reference in the given Target, if possible.
	Resolve(context.Context, *lsv1alpha1.Target) (*ResolvedTarget, error)
}

type ResolvedTarget struct {
	// Target contains the original target.
	*lsv1alpha1.Target

	// Resolved contains the content of the resolved reference.
	Resolved []byte
}

// NewResolvedTarget is a constructor for ResolvedTarget.
// Note that this type should usually come from a call to Resolve instead of being constructed manually.
func NewResolvedTarget(target *lsv1alpha1.Target) *ResolvedTarget {
	return &ResolvedTarget{
		Target: target,
	}
}

// Content returns the (potentially resolved) content of the Target.
// If the Target has a resolved reference, it returns Resolved. Otherwise, it returns the inline config of the Target.
func (rt *ResolvedTarget) Content() []byte {
	if rt.Resolved != nil {
		return rt.Resolved
	}
	return rt.Target.Spec.Configuration.RawMessage
}
