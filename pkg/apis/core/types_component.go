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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ComponentPhase string

const (
	ComponentPhaseInit        ComponentPhase = "Init"
	ComponentPhaseProgressing ComponentPhase = "Progressing"
	ComponentPhaseCompleted   ComponentPhase = "Completed"
	ComponentPhaseFailed      ComponentPhase = "Failed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ComponentList contains a list of Components
type ComponentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Component `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ComponentDefinition contains the configuration of a component
// +kubebuilder:subresource:status
type Component struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ComponentSpec   `json:"spec"`
	Status ComponentStatus `json:"status"`
}

type ComponentSpec struct {
	DefinitionRef string `json:"definitionRef"`

	Imports []Import `json:"imports"`
	Exports []Export `json:"exports"`
}

type ComponentStatus struct {
	Phase     ComponentPhase  `json:"phase,omitempty"`
	Executors []ExecutorState `json:"executors,omitempty"`
}

type Import struct {
	From     string  `json:"from"`
	To       string  `json:"to"`
	Type     string  `json:"type"`
	Required bool    `json:"required"`
	Default  Default `json:"default"`
}

type Export struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

type Default struct {
	Value     json.RawMessage `json:"value"`
	Reference string          `json:"ref"`
}

// ExecutorState tracks the state of a executor
type ExecutorState struct {
	Phase ComponentPhase `json:"phase"`

	// +optional
	Resource *v1.ObjectReference `json:"resource"`

	// Conditions contains the last observed conditions of the component.
	// +optional
	Conditions []Condition `json:"conditions,omitempty"`
}
