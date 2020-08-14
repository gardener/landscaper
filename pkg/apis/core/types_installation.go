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

package core

import (
	"encoding/json"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ComponentInstallationPhase string

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InstallationList contains a list of Components
type InstallationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Installation `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Blueprint contains the configuration of a component
// +kubebuilder:subresource:status
type Installation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstallationSpec   `json:"spec"`
	Status InstallationStatus `json:"status"`
}

// InstallationSpec defines a component installation.
type InstallationSpec struct {
	// BlueprintRef is the resolved reference to the definition.
	BlueprintRef InstallationBlueprintReference `json:"definitionRef"`

	// Imports define the import mapping for the referenced definition.
	// These values are by default auto generated from the parent definition.
	// +optional
	Imports []ImportMappingDefinition `json:"imports,omitempty"`

	// Exports define the export mappings for the referenced definition.
	// These values are by default auto generated from the parent definition.
	// todo: add static data boolean
	// +optional
	Exports []DefinitionExportMapping `json:"exports,omitempty"`

	// StaticData contains a list of data sources that are used to satisfy imports
	// +optional
	StaticData []StaticDataSource `json:"staticData,omitempty"`
}

// InstallationStatus contains the current status of a Installation.
type InstallationStatus struct {
	// Phase is the current phase of the installation.
	Phase ComponentInstallationPhase `json:"phase,omitempty"`

	// ObservedGeneration is the most recent generation observed for this ControllerInstallations.
	// It corresponds to the ControllerInstallations generation, which is updated on mutation by the landscaper.
	ObservedGeneration int64 `json:"observedGeneration"`

	// Conditions contains the actual condition of a installation
	Conditions []Condition `json:"conditions,omitempty"`

	// ConfigGeneration is the generation of the exported values.
	ConfigGeneration string `json:"configGeneration"`

	// ExportReference references the object that contains the exported values.
	ExportReference *ObjectReference `json:"exportRef,omitempty"`

	// ImportReference references the object that contains the temporary imported values.
	ImportReference *ObjectReference `json:"importRef,omitempty"`

	// Imports contain the state of the imported values.
	Imports []ImportState `json:"imports,omitempty"`

	// InstallationReferences contain all references to sub-components
	// that are created based on the component definition.
	InstallationReferences []NamedObjectReference `json:"installationRefs,omitempty"`

	// ExecutionReference is the reference to the execution that schedules the templated execution items.
	// +optional
	ExecutionReference *ObjectReference `json:"executionRef,omitempty"`
}

// InstallationBlueprintReference describes a reference to a blueprint defined by a component descriptor.
type InstallationBlueprintReference struct {
	VersionedResourceReference `json:",inline"`
	// RepositoryContext defines the context of the component repository to resolve blueprints.
	cdv2.RepositoryContext `json:",inline"`
}

// StaticDataSource defines a static data source.
type StaticDataSource struct {
	// Value defined inline a raw data
	// +optional
	Value json.RawMessage `json:"value,omitempty"`

	// ValueFrom defines data from an external resource
	ValueFrom *StaticDataValueFrom `json:"valueFrom,omitempty"`
}

// StaticDataValueFrom defines a static data that is read from a external resource.
type StaticDataValueFrom struct {
	// Selects a key of a secret in the installations's namespace
	// +optional
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef,omitempty"`

	// Selects a key from multiple secrets in the installations's namespace
	// that matches the given labels.
	// +optional
	SecretLabelSelector *SecretLabelSelectorRef `json:"secretLabelSelector,omitempty"`
}

// SecretLabelSelectorRef selects secrets with the given label and key.
type SecretLabelSelectorRef struct {
	// Selector is a map of labels to select specific secrets.
	Selector map[string]string `json:"selector"`

	// The key of the secret to select from.  Must be a valid secret key.
	Key string `json:"key"`
}

// ImportState hold the state of a import
type ImportState struct {
	// From is the from key of the import
	From string `json:"from"`

	// To is the to key of the import
	To string `json:"to"`

	// SourceRef is the reference to the installation where the value is imported
	SourceRef *ObjectReference `json:"sourceRef,omitempty"`

	// ConfigGeneration is the generation of the imported value.
	ConfigGeneration string `json:"configGeneration"`
}
