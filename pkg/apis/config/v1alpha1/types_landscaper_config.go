// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LandscaperConfiguration contains all configuration for the landscaper controllers
type LandscaperConfiguration struct {
	metav1.TypeMeta `json:",inline"`
	// DefaultOCI defines the default oci configuration which is used
	// if it's not overwritten by more specific configuration.
	DefaultOCI *OCIConfiguration `json:"defaultOCI,omitempty"`
	// Registries configures the landscaper registries.
	Registries RegistriesConfiguration `json:"registries"`
}

// RegistriesConfiguration contains the configuration options for blueprint and component registries
type RegistriesConfiguration struct {
	// Artifacts contains the configuration to fetch blueprints and jsonschemas
	// from local or remote registries.
	Artifacts RegistryConfiguration `json:"blueprints"`
	// Components contains the configuration for the used component descriptor registry.
	Components RegistryConfiguration `json:"components"`
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
	// ConfigPaths configures local file paths to look for resources.
	ConfigPaths []string `json:"configPaths"`
}

// OCIConfiguration holds configuration for the oci registry
type OCIConfiguration struct {
	// ConfigFiles path to additional docker configuration files
	// +optional
	ConfigFiles []string `json:"configFiles,omitempty"`

	// Cache holds configuration for the oci cache
	// +optional
	Cache *OCICacheConfiguration `json:"cache,omitempty"`
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
