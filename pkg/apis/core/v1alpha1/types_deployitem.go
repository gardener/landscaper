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

package v1alpha1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DeployItemList contains a list of DeployItems
type DeployItemList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeployItem `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DeployItem defines a DeployItem that should be processed by a external deployer
// +kubebuilder:resource:path="deployitems"
// +kubebuilder:resource:scope="Namespaced"
// +kubebuilder:resource:shortName="di,deploy"
// +kubebuilder:resource:singular="deployitem"
// +kubebuilder:subresource:status
type DeployItem struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeployItemSpec   `json:"spec"`
	Status DeployItemStatus `json:"status"`
}

// DeployItemSpec contains the definition of a deploy item.
type DeployItemSpec struct {
	// DataType is the type of the deployer that should handle the item.
	Type ExecutionType `json:"type"`

	// ImportReference is the reference to the object containing all imported values.
	ImportReference ObjectReference `json:"importRef,omitempty"`

	// Configuration contains the deployer type specific configuration.
	Configuration json.RawMessage `json:"config,omitempty"`
}

// DeployItemStatus contains the status of a deploy item
type DeployItemStatus struct {
	// Phase is the current phase of the DeployItem
	Phase ExecutionPhase `json:"phase,omitempty"`

	// Conditions contains the actual condition of a deploy item
	Conditions []Condition `json:"conditions,omitempty"`

	// ExportReference is the reference to the object that contains the exported values.
	// +optional
	ExportReference *ObjectReference `json:"exportRef,omitempty"`
}
