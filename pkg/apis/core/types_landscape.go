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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LandscapeConfigurationList contains a list of LandscapeConfiguration
type LandscapeConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LandscapeConfiguration `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LandscapeConfiguration defines the landscape configuration that consists of multiple secrets.
// Must be a singleton.
type LandscapeConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec LandscapeConfigurationSpec `json:"spec"`

	// +optional
	Status LandscapeConfigurationStatus `json:"status"`
}

// LandscapeConfigurationSpec contains the values for the Landscape generation
type LandscapeConfigurationSpec struct {
	SecretReferences []ObjectReference `json:"secretRefs,omitempty"`
}

// LandscapeConfigurationStatus contains the status of the landscape configuration
type LandscapeConfigurationStatus struct {
	// ObservedGeneration is the most recent generation observed for this landscapeConfiguration.
	// It corresponds to the LandscapeConfig generation, which is updated on mutation.
	ObservedGeneration int64 `json:"observedGeneration"`

	// Conditions contains the actual condition of a landscape config
	// +optional
	Conditions []Condition `json:"conditions,omitempty"`

	// ConfigGeneration is the generation of the exported values.
	ConfigGeneration int64 `json:"configGeneration"`

	// ExportReference references the object that contains the exported values.
	ConfigReference *ObjectReference `json:"configRef,omitempty"`

	// Secrets contains the status of the observed referenced secrets.
	// +optional
	Secrets []VersionedObjectReference `json:"secrets,omitempty"`
}
