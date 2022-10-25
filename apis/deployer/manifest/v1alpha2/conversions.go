// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	"k8s.io/apimachinery/pkg/conversion"

	"github.com/gardener/landscaper/apis/deployer/manifest"
)

// Convert_manifest_ProviderStatus_To_v1alpha2_ProviderStatus is a manual conversion function
func Convert_manifest_ProviderStatus_To_v1alpha2_ProviderStatus(in *manifest.ProviderStatus, out *ProviderStatus, _ conversion.Scope) error {
	if in.ManagedResources != nil {
		in.ManagedResources.DeepCopyInto(&out.ManagedResources)
	} else {
		out.ManagedResources = nil
	}
	return nil
}
