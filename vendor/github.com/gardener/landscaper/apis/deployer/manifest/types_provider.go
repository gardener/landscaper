// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProviderConfiguration is the helm deployer configuration that is expected in a DeployItem
type ProviderConfiguration struct {
	metav1.TypeMeta `json:",inline"`
	// Kubeconfig is the base64 encoded kubeconfig file.
	// By default the configured target is used to deploy the resources
	// +optional
	Kubeconfig string `json:"kubeconfig"`
	// UpdateStrategy defines the strategy how the manifest are updated in the cluster.
	// +optional
	UpdateStrategy UpdateStrategy `json:"updateStrategy"`
	// Manifests contains a list of manifests that should be applied in the target cluster
	Manifests []Manifest `json:"manifests,omitempty"`
}

// ManifestPolicy defines the strategy how a manifest should be managed
// by the deployer.
type ManifestPolicy string

const (
	// ManagePolicy is the default policy where the resource is
	// created, updated and deleted and it occupies already managed resources
	ManagePolicy ManifestPolicy = "manage"
	// FallbackPolicy defines a policy where the resource is created, updated and deleted
	// but only if not already managed by someone else (check for annotation with landscaper identity, deployitem name + namespace)
	FallbackPolicy ManifestPolicy = "fallback"
	// KeepPolicy defines a policy where the resource is only created and updated but not deleted.
	// It is not deleted when the whole deploy item is nor when the resource is not defined anymore.
	KeepPolicy ManifestPolicy = "keep"
	// IgnorePolicy defines a policy where the resource is completely ignored by the deployer.
	IgnorePolicy ManifestPolicy = "ignore"
)

// Manifest defines a manifest that is managed by the deployer.
type Manifest struct {
	// Policy defines the manage policy for that resource.
	Policy ManifestPolicy `json:"policy,omitempty"`
	// Manifest defines the raw k8s manifest.
	Manifest *runtime.RawExtension `json:"manifest,omitempty"`
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
	metav1.TypeMeta `json:",inline"`
	// ManagedResources contains all kubernetes resources that are deployed by the deployer.
	ManagedResources []ManagedResourceStatus `json:"managedResources,omitempty"`
}

// ManagedResourceStatus describes the managed resource and their metadata.
type ManagedResourceStatus struct {
	// Policy defines the manage policy for that resource.
	Policy ManifestPolicy `json:"policy,omitempty"`
	// Resources describes the managed kubernetes resource.
	Resource lsv1alpha1.TypedObjectReference `json:"resource"`
}
