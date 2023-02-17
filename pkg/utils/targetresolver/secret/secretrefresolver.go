// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package secret

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsutils "github.com/gardener/landscaper/pkg/utils"
	. "github.com/gardener/landscaper/pkg/utils/targetresolver"
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

func (srr SecretRefResolver) Resolve(ctx context.Context, target *lsv1alpha1.Target) (*lsv1alpha1.ResolvedTarget, error) {
	rt := NewResolvedTarget(target)

	if target.Spec.SecretRef != nil {
		sr := &lsv1alpha1.SecretReference{
			ObjectReference: lsv1alpha1.ObjectReference{
				Name:      target.Spec.SecretRef.Name,
				Namespace: target.Namespace,
			},
			Key: target.Spec.SecretRef.Key,
		}

		_, rawContent, _, err := lsutils.ResolveSecretReference(ctx, srr.Client, sr)
		if err != nil {
			return nil, err
		}
		rt.Content = string(rawContent)
	}

	return rt, nil
}
