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
	metav1.TypeMeta
	// Controllers contains all controller specific configuration.
	Controllers Controllers
	// RepositoryContext defines the default repository context that should be used to resolve component descriptors.
	// DEPRECATED: use controllers.context.config.default.repositoryContext instead.
	// +optional
	RepositoryContext *cdv2.UnstructuredTypedObject
	// Registry configures the landscaper registry to resolve component descriptors, blueprints and other artifacts.
	Registry RegistryConfiguration
	// BlueprintStore contains the configuration for the blueprint cache.
	BlueprintStore BlueprintStore
	// Metrics allows to configure how metrics are exposed
	//+optional
	Metrics *MetricsConfiguration
	// CrdManagement configures whether the landscaper controller should deploy the CRDs it needs into the cluster
	// +optional
	CrdManagement CrdManagementConfiguration
	// DeployerManagement configures the deployer management of the landscaper.
	// +optional
	DeployerManagement DeployerManagementConfiguration
	// DeployItemTimeouts contains configuration for multiple deploy item timeouts
	// +optional
	DeployItemTimeouts *DeployItemTimeouts
	// LsDeployments contains the names of the landscaper deployments
	// +optional
	LsDeployments *LsDeployments
}

// LsDeployments contains the names of the landscaper deployments.
type LsDeployments struct {
	LsController string
	WebHook      string
}

// CommonControllerConfig describes common controller configuration that can be included in
// the specific controller configurations.
type CommonControllerConfig struct {
	// Workers is the maximum number of concurrent Reconciles which can be run.
	// Defaults to 1.
	Workers int

	// CacheSyncTimeout refers to the time limit set to wait for syncing the kubernetes resource caches.
	// Defaults to 2 minutes if not set.
	CacheSyncTimeout *metav1.Duration
}

// Controllers contains all configuration for the specific controllers
type Controllers struct {
	// SyncPeriod determines the minimum frequency at which watched resources are
	// reconciled. A lower period will correct entropy more quickly, but reduce
	// responsiveness to change if there are many watched resources. Change this
	// value only if you know what you are doing. Defaults to 10 hours if unset.
	// there will a 10 percent jitter between the SyncPeriod of all controllers
	// so that all controllers will not send list requests simultaneously.
	//
	// This applies to all controllers.
	//
	// A period sync happens for two reasons:
	// 1. To insure against a bug in the controller that causes an object to not
	// be requeued, when it otherwise should be requeued.
	// 2. To insure against an unknown bug in controller-runtime, or its dependencies,
	// that causes an object to not be requeued, when it otherwise should be
	// requeued, or to be removed from the queue, when it otherwise should not
	// be removed.
	SyncPeriod *metav1.Duration
	// Installations contains the controller config that reconciles installations.
	Installations InstallationsController
	// Installations contains the controller config that reconciles executions.
	Executions ExecutionsController
	// DeployItems contains the controller config that reconciles deploy items.
	DeployItems DeployItemsController
	// Contexts contains the controller config that reconciles context objects.
	Contexts ContextsController
}

// InstallationsController contains the controller config that reconciles installations.
type InstallationsController struct {
	CommonControllerConfig
}

// ExecutionsController contains the controller config that reconciles executions.
type ExecutionsController struct {
	CommonControllerConfig
}

// DeployItemsController contains the controller config that reconciles deploy items.
type DeployItemsController struct {
	CommonControllerConfig
}

// ContextsController contains all configuration for the context controller.
type ContextsController struct {
	CommonControllerConfig
	Config ContextControllerConfig
}

// ContextControllerConfig contains the context specific configuration.
type ContextControllerConfig struct {
	Default ContextControllerDefaultConfig
}

// ContextControllerDefaultConfig contains the configuration for the context defaults.
type ContextControllerDefaultConfig struct {
	// Disable disables the default controller.
	// If disabled no default contexts are created.
	Disable bool
	// ExcludedNamespaces defines a list of namespaces where no default context should be created.
	// +optional
	ExcludedNamespaces []string
	// RepositoryContext defines the default repository context that should be used to resolve component descriptors.
	// +optional
	RepositoryContext *cdv2.UnstructuredTypedObject
}

// DeployItemTimeouts contains multiple timeout configurations for deploy items
type DeployItemTimeouts struct {
	// PickupTimeout defines how long a deployer can take to react on changes to a deploy item before the landscaper will mark it as failed.
	// Allowed values are 'none' (to disable pickup timeout detection) and anything that is understood by golang's time.ParseDuration method.
	// Defaults to five minutes if not specified.
	// +optional
	Pickup *lscore.Duration
	// Abort specifies how long the deployer may take to abort handling a deploy item after getting the abort annotation.
	// Allowed values are 'none' (to disable abort timeout detection) and anything that is understood by golang's time.ParseDuration method.
	// Defaults to five minutes if not specified.
	// +optional
	Abort *lscore.Duration
	// ProgressingDefault specifies how long the deployer may take to apply a deploy item by default. The value can be overwritten per deploy item in 'spec.timeout'.
	// Allowed values are 'none' (to disable abort timeout detection) and anything that is understood by golang's time.ParseDuration method.
	// Defaults to ten minutes if not specified.
	// +optional
	ProgressingDefault *lscore.Duration
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
	// DeployerRepositoryContext defines the repository context to fetch the component descriptors for the
	// default deployer (helm, container, manifest). If not set, the open source repository context is used.
	// +optional
	DeployerRepositoryContext *cdv2.UnstructuredTypedObject `json:"deployerRepositoryContext,omitempty"`
}

// LandscaperAgentConfiguration is the landscaper specific agent configuration
type LandscaperAgentConfiguration struct {
	// Disable disables the default agent that is started with the landscaper.
	// This is automatically disabled if the deployment management is disabled.
	Disable            bool `json:"disable"`
	AgentConfiguration `json:",inline"`
}

// IndexMethod describes the blueprint store index method
type IndexMethod string

const (
	// BlueprintDigestIndex describes a IndexMethod that uses the digest of the blueprint.
	// This is useful if blueprints and component descriptors are not immutable (e.g. during development)
	BlueprintDigestIndex IndexMethod = "BlueprintDigestIndex"
	// ComponentDescriptorIdentityMethod describes a IndexMethod that uses the component descriptor identity.
	// This means that the blueprint is uniquely identified using the component-descriptors repository, name and version
	// with the blueprint resource identity.
	ComponentDescriptorIdentityMethod IndexMethod = "ComponentDescriptorIdentityMethod"
)

// BlueprintStore contains the configuration for the blueprint store.
type BlueprintStore struct {
	// Path defines the root path where the blueprints are cached.
	Path string
	// DisableCache disables the cache and always fetches the blob from the registry.
	// The blueprint is still stored on the filesystem.
	DisableCache bool
	// IndexMethod describes the method that should be used to index blueprints in the store.
	// If component descriptors and blueprint are immutable (blueprints cannot be updated) use ComponentDescriptorIdentityMethod
	// otherwise use the BlueprintDigestIndex to index by the content hash.
	// Defaults to ComponentDescriptorIdentityMethod
	// +optional
	IndexMethod IndexMethod
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
