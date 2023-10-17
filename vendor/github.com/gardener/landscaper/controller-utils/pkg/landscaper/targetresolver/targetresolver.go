// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package targetresolver

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	genericresolver "github.com/gardener/landscaper/controller-utils/pkg/landscaper/targetresolver/generic"
)

type TargetResolver interface {
	// Resolve resolves the reference in the given Target, if possible.
	Resolve(context.Context, *lsv1alpha1.Target) (*lsv1alpha1.ResolvedTarget, error)
}

// Resolve is a generic resolve function for targets.
// It checks which target resolver to use, instantiates it and uses it to resolve the target.
// It therefore requires all arguments that are required for any of the contained targetresolvers.
// These arguments are only used if the corresponding resolver is actually used, so they can be nil for resolvers that are known to not be required.
// Internally, a GenericResolver is used (which uses the actual resolvers).
func Resolve(ctx context.Context, target *lsv1alpha1.Target, c client.Client) (*lsv1alpha1.ResolvedTarget, error) {
	return genericresolver.New(c).Resolve(ctx, target)
}
