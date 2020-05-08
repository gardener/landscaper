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

package core

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DeployItemList contains a list of DeployItems
type DeployItemList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Type `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DeployItem defines a DeployItem that should be processed by a external deployer
type DeployItem struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TypeSpec   `json:"spec"`
	Status TypeStatus `json:"status"`
}

type DeployItemSpec struct {
	Type string `json:"type"`
	Import SecretRef `json:"import,omitempty"`
	DeployConfig json.RawMessage `json:"deployConfig"`
}

type DeployItemStatus struct {
	Phase ComponentPhase `json:"phase,omitempty"`

	// +optional
	ExportGeneration int64 `json:"exportGeneration,omitempty"`

	// +optional
	Export *DeployItemExport `json:"export,omitempty"`
}

type DeployItemExport struct {
	// +optional
	Value string `json:"value,omitempty"`

	// +optional
	ValueRef *SecretRef `json:"valueRef,omitempty"`
}

