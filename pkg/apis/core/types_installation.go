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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ComponentInstallationPhase string

const (
	ComponentPhaseInit        ComponentInstallationPhase = "Init"
	ComponentPhaseWaitingDeps ComponentInstallationPhase = "WaitingDependencies"
	ComponentPhaseProgressing ComponentInstallationPhase = "Progressing"
	ComponentPhaseCompleted   ComponentInstallationPhase = "Completed"
	ComponentPhaseFailed      ComponentInstallationPhase = "Failed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ComponentInstallationList contains a list of Components
type ComponentInstallationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ComponentInstallation `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ComponentDefinition contains the configuration of a component
// +kubebuilder:subresource:status
type ComponentInstallation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ComponentInstallationSpec   `json:"spec"`
	Status ComponentInstallationStatus `json:"status"`
}

// ComponentInstallationSpec defines a component installation.
type ComponentInstallationSpec struct {
	// DefinitionRef is a reference to the component definition.
	DefinitionRef string `json:"definitionRef"`

	// Imports define the import mapping for the referenced definition.
	// These values are by default auto generated from the parent definition.
	// +optional
	Imports []DefinitionImportMapping `json:"imports,omitempty"`

	// Exports define the export mappings for the referenced definition.
	// These values are by default auto generated from the parent definition.
	// +optional
	Exports []DefinitionExportMapping `json:"exports,omitempty"`
}

// ComponentInstallationStatus contains the current status of a ComponentInstallation.
type ComponentInstallationStatus struct {
	// Phase is the current phase of the installation.
	Phase ComponentInstallationPhase `json:"phase,omitempty"`

	// ObservedGeneration is the most recent generation observed for this ControllerInstallations.
	// It corresponds to the ControllerInstallations generation, which is updated on mutation by the landscaper.
	ObservedGeneration int64 `json:"observedGeneration"`

	// Conditions contains the actual condition of a deploy item
	Conditions []Condition `json:"conditions,omitempty"`

	// ConfigGeneration is the generation of the exported values.
	ConfigGeneration int64 `json:"configGeneration"`

	// ExportReference references the object that contains the exported values.
	ExportReference ObjectReference `json:"exportReference,omitempty"`

	// Imports contain the state of the imported values.
	Imports []ImportState `json:"imports,omitempty"`

	// InstallationReferences contain all references to sub-components
	// that are created based on the component definition.
	InstallationReferences []ObjectReference `json:"installationRefs,omitempty"`

	// DeployItemReferences contain the state of all deploy items
	DeployItemReferences []ObjectReference `json:"deployItemRefs,omitempty"`
}

// ImportState hold the state of a import
type ImportState struct {
	// From is the from key of the import
	From string `json:"from"`

	// InstallationRef is the reference to the installation where the value is imported
	InstallationRef ObjectReference `json:"installationRef"`

	// ConfigGeneration is the generation of the imported value.
	ConfigGeneration int64 `json:"configGeneration"`
}
