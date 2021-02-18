// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

// Chart defines the helm chart to render and apply.
type Chart struct {
	// Ref defines the reference to a helm chart in a oci repository.
	// +optional
	Ref string `json:"ref,omitempty"`
	// FromResource fetches the chart based on the resource's access method.
	// The resource is defined as part of a component descriptor which is necessary to also handle
	// local artifacts.
	// +optional
	FromResource *RemoteChartReference `json:"fromResource,omitempty"`
	// Archive defines a compressed tarred helm chart as base64 encoded string.
	// +optional
	Archive *ArchiveAccess `json:"archive,omitempty"`
}

// RemoteChartReference defines a reference to a remote Helm chart through a Component-Descriptor
type RemoteChartReference struct {
	lsv1alpha1.ComponentDescriptorDefinition `json:",inline"`
	// ResourceName is the name of the Helm chart as defined by a component descriptor.
	ResourceName string `json:"resourceName"`
}

// ArchiveAccess defines the access for a helm chart as compressed archive.
type ArchiveAccess struct {
	// Raw defines a compressed tarred helm chart as base64 encoded string.
	// +optional
	Raw string `json:"raw,omitempty"`
	// Remote defines the remote access for a helm chart as compressed archive.
	// +optional
	Remote *RemoteArchiveAccess `json:"remote,omitempty"`
}

// RemoteArchiveAccess defines the remote access for a helm chart as compressed archive.
type RemoteArchiveAccess struct {
	// URL defines a compressed tarred helm chart that is fetched from a url.
	// +optional
	URL string `json:"url,omitempty"`
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
	metav1.TypeMeta `json:",inline"`

	// ManagedResources contains all kubernetes resources that are deployed by the helm deployer.
	ManagedResources []lsv1alpha1.TypedObjectReference `json:"managedResources,omitempty"`
}
