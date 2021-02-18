// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProviderConfiguration is the manifest deployer configuration that is expected in a DeployItem
type ProviderConfiguration struct {
	metav1.TypeMeta `json:",inline"`
	// Kubeconfig is the base64 encoded kubeconfig file.
	// By default the configured target is used to deploy the resources
	// +optional
	Kubeconfig string `json:"kubeconfig"`
	// UpdateStrategy defines the strategy how the manifest are updated in the cluster.
	// Defaults to "update".
	// +optional
	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`
	// HealthChecks condigures the health checks.
	// +optional
	HealthChecks HealthChecksConfiguration `json:"healthChecks,omitempty"`
	// DeleteTimeout is the time to wait before giving up on a resource to be deleted.
	// Defaults to 60s.
	// +optional
	DeleteTimeout string `json:"deleteTimeout,omitempty"`
	// Manifests contains a list of manifests that should be applied in the target cluster
	Manifests []*runtime.RawExtension `json:"manifests,omitempty"`
}

// UpdateStrategy defines the strategy that is used to apply resources to the cluster.
type UpdateStrategy string

const (
	UpdateStrategyUpdate UpdateStrategy = "update"
	UpdateStrategyPatch  UpdateStrategy = "patch"
)

// HealthChecksConfiguration contains the condiguration for health checks.
type HealthChecksConfiguration struct {
	// DisableDefault allows to disable the default health checks.
	// +optional
	DisableDefault bool `json:"disableDefault,omitempty"`
	// Timeout is the time to wait before giving up on a resource to be healthy.
	// Defaults to 60s.
	// +optional
	Timeout string `json:"timeout,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProviderStatus is the manifest provider specific status
type ProviderStatus struct {
	metav1.TypeMeta `json:",inline"`
	// ManagedResources contains all kubernetes resources that are deployed by the manifest deployer.
	ManagedResources []lsv1alpha1.TypedObjectReference `json:"managedResources,omitempty"`
}
