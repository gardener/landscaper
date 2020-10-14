// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	// Chart defines helm chart to be templated and applied.
	Chart Chart `json:"chart"`

	// Name is the release name of the chart
	Name string `json:"name"`

	// Namespace is the release namespace of the chart
	Namespace string `json:"namespace"`

	// Values are the values that are used for templating.
	Values json.RawMessage `json:"values,omitempty"`

	// ExportsFromManifests describe the exports from the templated manifests that should be exported by the helm deployer.
	// +optional
	ExportsFromManifests []ExportFromManifestItem `json:"exportsFromManifests,omitempty"`
}

// UpdateStrategy defines the strategy that is used to apply resources to the cluster.
type UpdateStrategy string

const (
	UpdateStrategyUpdate UpdateStrategy = "update"
	UpdateStrategyPatch  UpdateStrategy = "patch"
)

// Chart defines the helm chart to render and apply.
type Chart struct {
	// Ref defines the reference to a helm chart in a oci repository.
	// +optional
	Ref string `json:"ref,omitempty"`
	// Tar defines a tarred helm chart as base64 encoded string.
	// Either a chart or a tar has to be defined
	// +optional
	Tar string `json:"tar,omitempty"`
}

// ExportFromManifestItem describes one export that is read from the templates values or a templated resource.
// The value will be by default read from the values if fromResource is not specified.
type ExportFromManifestItem struct {
	// Key is the key that the value from JSONPath is exported to.
	Key string `json:"key"`

	// JSONPath is the jsonpath to look for a value.
	// The JSONPath root is the referenced resource
	JSONPath string `json:"jsonPath"`

	// FromResource specifies the name of the resource where the value should be read.
	FromResource *lsv1alpha1.TypedObjectReference `json:"fromResource,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProviderStatus is the helm provider specific status
type ProviderStatus struct {
	metav1.TypeMeta

	// ManagedResources contains all kubernetes resources that are deployed by the helm deployer.
	ManagedResources []lsv1alpha1.TypedObjectReference `json:"managedResources,omitempty"`
}
