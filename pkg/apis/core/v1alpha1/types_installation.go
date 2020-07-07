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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EncompassedByLabel is the label that contains the name of the parent installation
// that encompasses the current installation.
// todo: add conversion
const EncompassedByLabel = "landscaper.gardener.cloud/encompassed-by"

// todo: keep only subinstallations?
const KeepChildrenAnnotation = "landscaper.gardener.cloud/keep-children"

// EnsureSubInstallationsCondition is the Conditions type to indicate the sub installation status.
const EnsureSubInstallationsCondition ConditionType = "EnsureSubInstallations"

// EnsureExecutionsCondition is the Conditions type to indicate the executions status.
const EnsureExecutionsCondition ConditionType = "EnsureExecutions"

// ValidateExportCondition is the Conditions type to indicate validation status of teh exported data.
const ValidateExportCondition ConditionType = "ValidateExport"

type ComponentInstallationPhase string

const (
	ComponentPhaseInit        ComponentInstallationPhase = "Init"
	ComponentPhasePending     ComponentInstallationPhase = "PendingDependencies"
	ComponentPhaseProgressing ComponentInstallationPhase = "Progressing"
	ComponentPhaseAborted     ComponentInstallationPhase = "Aborted"
	ComponentPhaseSucceeded   ComponentInstallationPhase = "Succeeded"
	ComponentPhaseFailed      ComponentInstallationPhase = "Failed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InstallationList contains a list of Components
type InstallationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Installation `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ComponentDefinition contains the configuration of a component
// +kubebuilder:resource:path="installations",scope="Namespaced",shortName="inst",singular="installation"
// +kubebuilder:subresource:status
type Installation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec InstallationSpec `json:"spec"`

	// +optional
	Status InstallationStatus `json:"status"`
}

// InstallationSpec defines a component installation.
type InstallationSpec struct {
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
	ConfigGeneration int64 `json:"configGeneration"`

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
	ExecutionReference *ObjectReference `json:"executionRefs,omitempty"`
}

// ImportState hold the state of a import.
type ImportState struct {
	// From is the from key of the import
	From string `json:"from"`

	// To is the to key of the import
	To string `json:"to"`

	// SourceRef is the reference to the installation where the value is imported
	SourceRef *TypedObjectReference `json:"sourceRef,omitempty"`

	// ConfigGeneration is the generation of the imported value.
	ConfigGeneration int64 `json:"configGeneration"`
}
