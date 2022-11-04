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
	Resolve(context.Context, *lsv1alpha1.Target) (*lsv1alpha1.ResolvedTarget, error)
}

// NewResolvedTarget is a constructor for ResolvedTarget.
// It puts the target's inline configuration into the Content field, if the target doesn't contain a secret reference.
func NewResolvedTarget(target *lsv1alpha1.Target) *lsv1alpha1.ResolvedTarget {
	res := &lsv1alpha1.ResolvedTarget{
		Target: target,
	}
	if target.Spec.SecretRef == nil && target.Spec.Configuration != nil {
		res.Content = string(target.Spec.Configuration.RawMessage)
	}
	return res
}
