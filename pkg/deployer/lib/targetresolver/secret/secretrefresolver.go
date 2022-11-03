// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package secret

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	. "github.com/gardener/landscaper/pkg/deployer/lib/targetresolver"
	lsutils "github.com/gardener/landscaper/pkg/utils"
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
		if len(target.Spec.SecretRef.Namespace) > 0 && target.Spec.SecretRef.Namespace != target.Namespace {
			return nil, fmt.Errorf("namespace of secret ref %s differs from target namespace %s", target.Spec.SecretRef.Namespace, target.Namespace)
		}

		_, rawContent, _, err := lsutils.ResolveSecretReference(ctx, srr.Client, target.Spec.SecretRef)
		if err != nil {
			return nil, err
		}
		rt.Content = string(rawContent)
	}

	return rt, nil
}
