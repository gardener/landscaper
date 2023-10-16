// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package generic

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/landscaper/targetresolver/secret"
)

// GenericResolver is a generic targetresolver that checks which actual resolver is required and then uses it to resolve the Target.
type GenericResolver struct {
	Client client.Client
}

// New creates a new GenericResolver.
// This constructor's argument list is the union of all actual targetresolver's arguments.
// The given arguments may be nil, if it is known that the specific resolver which requires the argument will not be needed,
// but this will cause errors if done wrong (which tries to instantiate an actual resolver with nil arguments).
func New(c client.Client) *GenericResolver {
	return &GenericResolver{
		Client: c,
	}
}

func (gr GenericResolver) Resolve(ctx context.Context, target *lsv1alpha1.Target) (*lsv1alpha1.ResolvedTarget, error) {
	var rt *lsv1alpha1.ResolvedTarget
	var err error
	if target.Spec.SecretRef != nil {
		if gr.Client == nil {
			return nil, fmt.Errorf("target contains a secret reference, but secretresolver cannot be constructed because given client is nil")
		}
		sr := secret.New(gr.Client)
		rt, err = sr.Resolve(ctx, target)
		if err != nil {
			return nil, fmt.Errorf("error resolving secret reference (%s/%s#%s) for Target '%s/%s': %w", target.Namespace, target.Spec.SecretRef.Name, target.Spec.SecretRef.Key, target.Namespace, target.Name, err)
		}
	} else {
		rt = lsv1alpha1.NewResolvedTarget(target)
	}
	return rt, nil
}
