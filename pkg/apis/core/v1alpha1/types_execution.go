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

type ExecutionPhase string

const (
	ExecutionPhaseInit        ExecutionPhase = "Init"
	ExecutionPhaseProgressing ExecutionPhase = "Progressing"
	ExecutionPhaseCompleted   ExecutionPhase = "Completed"
	ExecutionPhaseFailed      ExecutionPhase = "Failed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExecutionList contains a list of Executions‚
type ExecutionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Execution `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Execution contains the configuration of a execution and deploy item
// +kubebuilder:resource:path="executions"
// +kubebuilder:resource:scope="Namespaced"
// +kubebuilder:resource:shortName="exec"
// +kubebuilder:resource:singular="execution"
// +kubebuilder:subresource:status
type Execution struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExecutionSpec   `json:"spec"`
	Status ExecutionStatus `json:"status"`
}

// ExecutionSpec defines a execution plan.
type ExecutionSpec struct {
	// ImportReference is the reference to the object containing all imported values.
	ImportReference ObjectReference `json:"importRef,omitempty"`

	// Executions defines all execution items that need to be scheduled.
	Executions []ExecutionItem `json:"executions"`
}

// ExecutionStatus contains the current status of a execution.
type ExecutionStatus struct {
	// Phase is the current phase of the execution .
	Phase ExecutionPhase `json:"phase,omitempty"`

	// ObservedGeneration is the most recent generation observed for this Execution.
	// It corresponds to the Execution generation, which is updated on mutation by the landscaper.
	ObservedGeneration int64 `json:"observedGeneration"`

	// Conditions contains the actual condition of a execution
	Conditions []Condition `json:"conditions,omitempty"`

	// ExportReference references the object that contains the exported values.
	ExportReference ObjectReference `json:"exportReference,omitempty"`

	// DeployItemReferences contain the state of all deploy items
	DeployItemReferences []NamedObjectReference `json:"deployItemRefs,omitempty"`
}

// ExecutionItem defines a execution element that is translated into a deploy item.
type ExecutionItem struct {
	// Name is the unique name of the execution.
	Name string `json:"name"`

	// DataType is the DeployItem type of the execution
	Type ExecutionType `json:"type"`

	// Configuration contains the type specific configuration for the execution.
	Configuration json.RawMessage `json:"config"`
}
