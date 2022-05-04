// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	cr "github.com/gardener/landscaper/apis/deployer/utils/continuousreconcile"
	health "github.com/gardener/landscaper/apis/deployer/utils/readinesschecks"
)

const HelmChartRepoCredentialsKey = "helmChartRepoCredentials"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProviderConfiguration is the helm deployer configuration that is expected in a DeployItem
type ProviderConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// Kubeconfig is the base64 encoded kubeconfig file.
	// By default the configured target is used to deploy the resources
	// +optional
	Kubeconfig string `json:"kubeconfig"`

	// UpdateStrategy defines the strategy how the manifests are updated in the cluster.
	// Defaults to "update".
	// +optional
	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`

	// ReadinessChecks configures the readiness checks.
	// +optional
	ReadinessChecks health.ReadinessCheckConfiguration `json:"readinessChecks,omitempty"`

	// DeleteTimeout is the time to wait before giving up on a resource to be deleted.
	// Defaults to 180s.
	// +optional
	DeleteTimeout *lsv1alpha1.Duration `json:"deleteTimeout,omitempty"`

	// Chart defines helm chart to be templated and applied.
	Chart Chart `json:"chart"`

	// Name is the release name of the chart
	Name string `json:"name"`

	// Namespace is the release namespace of the chart
	Namespace string `json:"namespace"`

	// CreateNamespace configures the deployer to create the release namespace if not present.
	// The behavior is similar to the "helm install --create-namespace"
	CreateNamespace bool `json:"createNamespace"`

	// Values are the values that are used for templating.
	Values json.RawMessage `json:"values,omitempty"`

	// ExportsFromManifests describe the exports from the templated manifests that should be exported by the helm deployer.
	// +optional
	// DEPRECATED
	ExportsFromManifests []managedresource.Export `json:"exportsFromManifests,omitempty"`

	// Exports describe the exports from the templated manifests that should be exported by the helm deployer.
	// +optional
	Exports *managedresource.Exports `json:"exports,omitempty"`

	// ContinuousReconcile contains the schedule for continuous reconciliation.
	// +optional
	ContinuousReconcile *cr.ContinuousReconcileSpec `json:"continuousReconcile,omitempty"`

	// HelmDeployment indicates that helm is used as complete deployment mechanism and not only helm templating.
	// Default is true.
	// +optional
	HelmDeployment *bool `json:"helmDeployment,omitempty"`

	// HelmDeploymentConfig contains settings for helm operations. Only relevant if HelmDeployment is true.
	// +optional
	HelmDeploymentConfig *HelmDeploymentConfiguration `json:"helmDeploymentConfig,omitempty"`
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
	// FromResource fetches the chart based on the resource's access method.
	// The resource is defined as part of a component descriptor which is necessary to also handle
	// local artifacts.
	// +optional
	FromResource *RemoteChartReference `json:"fromResource,omitempty"`
	// Archive defines a compressed tarred helm chart as base64 encoded string.
	// +optional
	Archive *ArchiveAccess `json:"archive,omitempty"`
	// HelmChartRepo defines a reference to a chart in a helm chart repo.
	// +optional
	HelmChartRepo *HelmChartRepo `json:"helmChartRepo,omitempty"`
}

// HelmChartRepo defines a reference to a chart in a helm chart repo
type HelmChartRepo struct {
	HelmChartRepoUrl string `json:"helmChartRepoUrl,omitempty"`
	HelmChartName    string `json:"helmChartName,omitempty"`
	HelmChartVersion string `json:"helmChartVersion,omitempty"`
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

// HelmDeploymentConfiguration defines settings for a helm deployment.
type HelmDeploymentConfiguration struct {
	Install   map[string]lsv1alpha1.AnyJSON `json:"install,omitempty"`
	Upgrade   map[string]lsv1alpha1.AnyJSON `json:"upgrade,omitempty"`
	Uninstall map[string]lsv1alpha1.AnyJSON `json:"uninstall,omitempty"`
}

// HelmInstallConfiguration defines settings for a helm install operation.
type HelmInstallConfiguration struct {
	Atomic bool `json:"atomic,omitempty"`
	// Timeout is the timeout for the operation in minutes.
	// +optional
	Timeout *lsv1alpha1.Duration `json:"timeout,omitempty"`
}

// HelmUpgradeConfiguration defines settings for a helm upgrade operation.
type HelmUpgradeConfiguration = HelmInstallConfiguration

// HelmUninstallConfiguration defines settings for a helm uninstall operation.
type HelmUninstallConfiguration struct {
	// Timeout is the timeout for the operation in minutes.
	// +optional
	Timeout *lsv1alpha1.Duration `json:"timeout,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProviderStatus is the helm provider specific status
type ProviderStatus struct {
	metav1.TypeMeta `json:",inline"`

	// ManagedResources contains all kubernetes resources that are deployed by the helm deployer.
	ManagedResources managedresource.ManagedResourceStatusList `json:"managedResources,omitempty"`
}

// HelmChartRepoCredentials contains the credentials to access hepl chart repos
type HelmChartRepoCredentials struct {
	Auths []Auth `json:"auths,omitempty"`
}

// Auth contains authentication info for one helm chart repo
type Auth struct {
	// URL contains URL of helm chart repo
	URL string `json:"url,omitempty"`
	// CustomCAData allows you to add a custom CA, which is useful if your server speaks HTTPS with a self-
	// signed certificate. The added certificate must be in PEM format and base64 encoded.
	// +optional
	CustomCAData string `json:"customCAData,omitempty"`
	// AuthHeader contains the value that will be set in the "Authorization" header when fetching the Chart, e.g.
	// "Basic dX...3dvcmQ=".
	AuthHeader string `json:"authHeader,omitempty"`
}
