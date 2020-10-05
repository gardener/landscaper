// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package v1alpha1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InstallationTemplate defines a subinstallation in a blueprint.
// +kubebuilder:skip
type InstallationTemplate struct {
	metav1.TypeMeta

	// Name is the unique name of the step
	Name string `json:"name"`

	// Reference defines a reference to a Blueprint.
	// The blueprint can reside in an OCI or other supported location.
	Blueprint InstallationTemplateBlueprintDefinition `json:"blueprint"`

	// Imports define the imported data objects and targets.
	// +optional
	Imports *InstallationImports `json:"imports,omitempty"`

	// ImportDataMappings contains a template for restructuring imports.
	// It is expected to contain a key for every blueprint-defined data import.
	// Missing keys will be defaulted to their respective data import.
	// Example: namespace: (( installation.imports.namespace ))
	// +optional
	ImportDataMappings map[string]json.RawMessage `json:"importDataMappings,omitempty"`

	// Exports define the exported data objects and targets.
	// +optional
	Exports *InstallationExports `json:"exports,omitempty"`

	// ExportDataMappings contains a template for restructuring exports.
	// It is expected to contain a key for every blueprint-defined data export.
	// Missing keys will be defaulted to their respective data export.
	// Example: namespace: (( blueprint.exports.namespace ))
	// +optional
	ExportDataMappings map[string]json.RawMessage `json:"exportDataMappings,omitempty"`
}

// InstallationTemplateBlueprintDefinition contains either a reference to a blueprint or an inline definition.
type InstallationTemplateBlueprintDefinition struct {
	// Ref is a reference to a blueprint.
	// Only blueprints that are defined by the component descriptor of the current blueprint can be referenced here.
	// Example: cd://componentReference/dns/localResources/blueprint
	// +optional
	Ref string `json:"ref,omitempty"`

	// Filesystem defines a virtual filesystem with all files needed for a blueprint.
	// The filesystem must be a YAML filesystem.
	// +optional
	Filesystem json.RawMessage `json:"filesystem,omitempty"`
}
