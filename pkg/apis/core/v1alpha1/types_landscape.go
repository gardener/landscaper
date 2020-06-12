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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CollectReferencedConfiguration is the Conditions type to indicate the status of the LandscapeConfig referenced configs collection.
const CollectReferencedConfiguration ConditionType = "CollectReferencedConfiguration"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LandscapeConfiguration defines the landscape configuration that consists of multiple secrets.
// Must be a singleton.
// +kubebuilder:subresource:status
type LandscapeConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LandscapeConfigurationSpec   `json:"spec"`
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
	Conditions []Condition `json:"conditions,omitempty"`

	// ConfigGeneration is the generation of the exported values.
	ConfigGeneration int64 `json:"configGeneration"`

	// ExportReference references the object that contains the exported values.
	ConfigReference *ObjectReference `json:"configRef,omitempty"`
}
