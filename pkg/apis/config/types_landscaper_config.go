// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LandscaperConfiguration contains all configuration for the landscaper controllers
type LandscaperConfiguration struct {
	metav1.TypeMeta `json:",inline"`
	// Registries configures the landscaper registries.
	Registries RegistriesConfiguration `json:"registries"`
}

// RegistriesConfiguration contains the configuration options for blueprint and component registries
type RegistriesConfiguration struct {
	// Blueprints contains the configuration for the used blueprint
	Blueprints RegistryConfiguration `json:"blueprints"`
	// Blueprints contains the configuration for the used component descriptor registry
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
