// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// ManagedInstanceLabel describes label that is added to every manifest deployer managed resource
// to define its corresponding instance.
const ManagedInstanceLabel = "manifest.deployer.landscaper.gardener.cloud/instance"

// ManagedDeployItemLabel describes label that is added to every manifest deployer managed resource
// to define its source deploy item.
const ManagedDeployItemLabel = "manifest.deployer.landscaper.gardener.cloud/deployitem"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Configuration is the manifest deployer configuration that configures the controller.
type Configuration struct {
	metav1.TypeMeta `json:",inline"`
	// TargetSelector describes all selectors the deployer should depend on.
	TargetSelector []lsv1alpha1.TargetSelector `json:"targetSelector,omitempty"`
}
