// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProviderConfiguration is the helm deployer configuration that is expected in a DeployItem
type ProviderConfiguration struct {
	metav1.TypeMeta
	// Kubeconfig is the base64 encoded kubeconfig file.
	// By default the configured target is used to deploy the resources
	// +optional
	Kubeconfig string `json:"kubeconfig"`
	// UpdateStrategy defines the strategy how the manifest are updated in the cluster.
	// Defaults to "update".
	// +optional
	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`
	// Manifests contains a list of manifests that should be applied in the target cluster
	Manifests []*runtime.RawExtension `json:"manifests,omitempty"`
}

// UpdateStrategy defines the strategy that is used to apply resources to the cluster.
type UpdateStrategy string

const (
	UpdateStrategyUpdate UpdateStrategy = "update"
	UpdateStrategyPatch  UpdateStrategy = "patch"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProviderStatus is the helm provider specific status
type ProviderStatus struct {
	metav1.TypeMeta
	// ManagedResources contains all kubernetes resources that are deployed by the helm deployer.
	ManagedResources []lsv1alpha1.TypedObjectReference `json:"managedResources,omitempty"`
}
