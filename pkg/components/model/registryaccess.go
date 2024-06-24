// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"context"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model/componentoverwrites"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

type RegistryAccess interface {
	GetComponentVersion(ctx context.Context, compKey *types.ComponentVersionKey) (ComponentVersion, error)

	//VerifySignature calls the ocm lib to verify the named signature in the component version with the public key or ca cert data.
	VerifySignature(componentVersion ComponentVersion, name string, pkeyData []byte, caCertData []byte) error
}

// GetComponentVersionWithOverwriter is like registryAccess.GetComponentVersion, but applies the given overwrites first.
func GetComponentVersionWithOverwriter(ctx context.Context,
	registryAccess RegistryAccess,
	cdRef *lsv1alpha1.ComponentDescriptorReference,
	overwriter componentoverwrites.Overwriter) (ComponentVersion, error) {

	if overwriter != nil {
		overwriter.Replace(cdRef)
	}

	return registryAccess.GetComponentVersion(ctx, &types.ComponentVersionKey{
		Name:    cdRef.ComponentName,
		Version: cdRef.Version,
	})
}
