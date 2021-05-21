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
	RepositoryContext *cdv2.RepositoryContext `json:"repositoryContext,omitempty"`
	// Registry configures the landscaper registry to resolve component descriptors, blueprints and other artifacts.
	Registry RegistryConfiguration `json:"registry"`
	// Metrics allows to configure how metrics are exposed
	//+optional
	Metrics *MetricsConfiguration `json:"metrics,omitempty"`
	// CrdManagement configures whether the landscaper controller should deploy the CRDs it needs into the cluster
	// +optional
	CrdManagement CrdManagementConfiguration `json:"crdManagement,omitempty"`
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
