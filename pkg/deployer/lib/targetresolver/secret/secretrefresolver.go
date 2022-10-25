// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package secret

import (
	"context"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	. "github.com/gardener/landscaper/pkg/deployer/lib/targetresolver"
	lsutils "github.com/gardener/landscaper/pkg/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ TargetResolver = SecretRefResolver{}

type SecretRefResolver struct {
	Client client.Client
}

func New(c client.Client) *SecretRefResolver {
	return &SecretRefResolver{
		Client: c,
	}
}

func (srr SecretRefResolver) Resolve(ctx context.Context, target *lsv1alpha1.Target) (*ResolvedTarget, error) {
	rt := NewResolvedTarget(target)

	if target.Spec.SecretRef != nil {
		var err error
		_, rt.Resolved, _, err = lsutils.ResolveSecretReference(ctx, srr.Client, target.Spec.SecretRef)
		if err != nil {
			return nil, err
		}
	}

	return rt, nil
}
