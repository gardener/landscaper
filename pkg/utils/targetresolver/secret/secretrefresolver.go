// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package secret

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"
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

func (srr SecretRefResolver) GetKubeconfigFromTarget(ctx context.Context, target *lsv1alpha1.Target) ([]byte, error) {
	resolvedTarget, err := srr.Resolve(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("target resolver: failed to resolve target")
	}

	targetConfig := &targettypes.KubernetesClusterTargetConfig{}
	err = yaml.Unmarshal([]byte(resolvedTarget.Content), targetConfig)
	if err != nil {
		return nil, fmt.Errorf("target resolver: failed to unmarshal target config: %w", err)
	}
	if targetConfig.Kubeconfig.StrVal == nil {
		return nil, fmt.Errorf("target resolver: target config contains no kubeconfig: %w", err)
	}

	kubeconfigBytes := []byte(*targetConfig.Kubeconfig.StrVal)
	return kubeconfigBytes, nil
}
