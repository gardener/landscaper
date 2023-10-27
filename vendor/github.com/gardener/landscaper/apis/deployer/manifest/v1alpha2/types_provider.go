// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cr "github.com/gardener/landscaper/apis/deployer/utils/continuousreconcile"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	health "github.com/gardener/landscaper/apis/deployer/utils/readinesschecks"
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
	// ReadinessChecks configures the readiness checks.
	// +optional
	ReadinessChecks health.ReadinessCheckConfiguration `json:"readinessChecks,omitempty"`
	// Manifests contains a list of manifests that should be applied in the target cluster
	Manifests []managedresource.Manifest `json:"manifests,omitempty"`
	// Exports describe the exports from the templated manifests that should be exported by the helm deployer.
	// +optional
	Exports *managedresource.Exports `json:"exports,omitempty"`
	// ContinuousReconcile contains the schedule for continuous reconciliation.
	// +optional
	ContinuousReconcile *cr.ContinuousReconcileSpec `json:"continuousReconcile,omitempty"`
}

// UpdateStrategy defines the strategy that is used to apply resources to the cluster.
type UpdateStrategy string

const (
	UpdateStrategyUpdate         UpdateStrategy = "update"
	UpdateStrategyPatch          UpdateStrategy = "patch"
	UpdateStrategyMerge          UpdateStrategy = "merge"
	UpdateStrategyMergeOverwrite UpdateStrategy = "mergeOverwrite"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProviderStatus is the manifest provider specific status
type ProviderStatus struct {
	metav1.TypeMeta `json:",inline"`
	// ManagedResources contains all kubernetes resources that are deployed by the deployer.
	ManagedResources managedresource.ManagedResourceStatusList `json:"managedResources,omitempty"`
}
