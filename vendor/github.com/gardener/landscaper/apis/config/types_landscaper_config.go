// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lscore "github.com/gardener/landscaper/apis/core"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LandscaperConfiguration contains all configuration for the landscaper controllers
type LandscaperConfiguration struct {
	metav1.TypeMeta `json:",inline"`
	// RepositoryContext defines the default repository context that should be used to resolve component descriptors.
	// +optional
	RepositoryContext *cdv2.UnstructuredTypedObject `json:"repositoryContext,omitempty"`
	// Registry configures the landscaper registry to resolve component descriptors, blueprints and other artifacts.
	Registry RegistryConfiguration `json:"registry"`
	// BlueprintStore contains the configuration for the blueprint cache.
	BlueprintStore BlueprintStore `json:"blueprintStore"`
	// Metrics allows to configure how metrics are exposed
	//+optional
	Metrics *MetricsConfiguration `json:"metrics,omitempty"`
	// CrdManagement configures whether the landscaper controller should deploy the CRDs it needs into the cluster
	// +optional
	CrdManagement CrdManagementConfiguration `json:"crdManagement,omitempty"`
	// DeployerManagement configures the deployer management of the landscaper.
	// +optional
	DeployerManagement DeployerManagementConfiguration `json:"deployerManagement,omitempty"`
	// DeployItemTimeouts contains configuration for multiple deploy item timeouts
	// +optional
	DeployItemTimeouts *DeployItemTimeouts `json:"deployItemTimeouts,omitempty"`
}

// DeployItemTimeouts contains multiple timeout configurations for deploy items
type DeployItemTimeouts struct {
	// PickupTimeout defines how long a deployer can take to react on changes to a deploy item before the landscaper will mark it as failed.
	// Allowed values are 'none' (to disable pickup timeout detection) and anything that is understood by golang's time.ParseDuration method.
	// Defaults to five minutes if not specified.
	// +optional
	Pickup *lscore.Duration `json:"pickup,omitempty"`
	// Abort specifies how long the deployer may take to abort handling a deploy item after getting the abort annotation.
	// Allowed values are 'none' (to disable abort timeout detection) and anything that is understood by golang's time.ParseDuration method.
	// Defaults to five minutes if not specified.
	// +optional
	Abort *lscore.Duration `json:"abort,omitempty"`
	// ProgressingDefault specifies how long the deployer may take to apply a deploy item by default. The value can be overwritten per deploy item in 'spec.timeout'.
	// Allowed values are 'none' (to disable abort timeout detection) and anything that is understood by golang's time.ParseDuration method.
	// Defaults to ten minutes if not specified.
	// +optional
	ProgressingDefault *lscore.Duration `json:"progressingDefault,omitempty"`
}

// RegistryConfiguration contains the configuration for the used definition registry
type RegistryConfiguration struct {
	// Local defines a local registry to use for definitions
	// +optional
	Local *LocalRegistryConfiguration `json:"local,omitempty"`

	// OCI defines a oci registry to use for definitions
	// +optional
	OCI *OCIConfiguration `json:"oci,omitempty"`
}

// LocalRegistryConfiguration contains the configuration for a local registry
type LocalRegistryConfiguration struct {
	// RootPath configures the root path of a local registry.
	// This path is used to search for components locally.
	RootPath string `json:"rootPath"`
}

// OCIConfiguration holds configuration for the oci registry
type OCIConfiguration struct {
	// ConfigFiles path to additional docker configuration files
	// +optional
	ConfigFiles []string `json:"configFiles,omitempty"`

	// Cache holds configuration for the oci cache
	// +optional
	Cache *OCICacheConfiguration `json:"cache,omitempty"`

	// AllowPlainHttp allows the fallback to http if https is not supported by the registry.
	AllowPlainHttp bool `json:"allowPlainHttp"`
	// InsecureSkipVerify skips the certificate validation of the oci registry
	InsecureSkipVerify bool `json:"insecureSkipVerify"`
}

// OCICacheConfiguration contains the configuration for the oci cache
type OCICacheConfiguration struct {
	// UseInMemoryOverlay enables an additional in memory overlay cache of oci images
	// +optional
	UseInMemoryOverlay bool `json:"useInMemoryOverlay,omitempty"`

	// Path specifies the path to the oci cache on the filesystem.
	// Defaults to /tmp/ocicache
	// +optional
	Path string `json:"path"`
}

// MetricsConfiguration allows to configure how metrics are exposed
type MetricsConfiguration struct {
	// Port specifies the port on which metrics are published
	Port int32 `json:"port"`
}

// CrdManagementConfiguration contains the configuration of the CRD management
type CrdManagementConfiguration struct {
	// DeployCustomResourceDefinitions specifies if CRDs should be deployed
	DeployCustomResourceDefinitions *bool `json:"deployCrd"`

	// ForceUpdate specifies whether existing CRDs should be updated
	// +optional
	ForceUpdate *bool `json:"forceUpdate,omitempty"`
}

// DeployerManagementConfiguration contains the configuration of the deployer management
type DeployerManagementConfiguration struct {
	// Disable disables the landscaper deployer management.
	Disable bool `json:"disable"`
	// Namespace defines the system namespace where the deployer installation should be deployed to.
	Namespace string `json:"namespace"`
	// Agent contains the landscaper agent configuration.
	Agent LandscaperAgentConfiguration `json:"agent"`
}

// LandscaperAgentConfiguration is the landscaper specific agent configuration
type LandscaperAgentConfiguration struct {
	// Disable disables the default agent that is started with the landscaper.
	// This is automatically disabled if the deployment management is disabled.
	Disable            bool `json:"disable"`
	AgentConfiguration `json:",inline"`
}

// BlueprintStore contains the configuration for the blueprint store.
type BlueprintStore struct {
	// Path defines the root path where the blueprints are cached.
	Path string `json:"path"`
	// DisableCache disables the cache and always fetches the blob from the registry.
	// The blueprint is still stored on the filesystem.
	DisableCache bool `json:"disableCache"`
	GarbageCollectionConfiguration
}

// GarbageCollectionConfiguration contains all options for the cache garbage collection.
type GarbageCollectionConfiguration struct {
	// Size is the size of the filesystem.
	// If the value is 0 there is no limit and no garbage collection will happen.
	// See the kubernetes quantity docs for detailed description of the format
	// https://github.com/kubernetes/apimachinery/blob/master/pkg/api/resource/quantity.go
	Size string
	// GCHighThreshold defines the percent of disk usage which triggers files garbage collection.
	GCHighThreshold float64
	// GCLowThreshold defines the percent of disk usage to which files garbage collection attempts to free.
	GCLowThreshold float64
	// ResetInterval defines the interval when the hit reset should run.
	ResetInterval metav1.Duration
	// PreservedHitsProportion defines the percent of hits that should be preserved.
	PreservedHitsProportion float64
}
