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

// TargetType defines the type of the target.
type TargetType string

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TargetList contains a list of Targets
type TargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Target `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Target defines a specific data object that defines target environment.
// Every deploy item can have a target which is used by the deployer to install the specific application.
// +kubebuilder:resource:path="targets",scope="Namespaced",shortName={"tg","tgt"},singular="target"
// +kubebuilder:printcolumn:JSONPath=".spec.type",name=Type,type=string
// +kubebuilder:printcolumn:JSONPath=`.metadata.labels['data\.landscaper\.gardener\.cloud\/context']`,name=Context,type=string
// +kubebuilder:printcolumn:JSONPath=`.metadata.labels['data\.landscaper\.gardener\.cloud\/key']`,name=Key,type=string
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=Age,type=date
type Target struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TargetSpec `json:"spec"`
}

// TargetSpec contains the definition of a target.
type TargetSpec struct {
	// Type is the type of the target that defines its data structure.
	// The actual schema may be defined by a target type crd in the future.
	Type TargetType `json:"type"`
	// Configuration contains the target type specific configuration.
	// +optional
	// +kubebuilder:validation:XPreserveUnknownFields
	Configuration json.RawMessage `json:"config,omitempty"`
}

// TargetTemplate exposes specific parts of a target that are used in the exports
// to export a target
type TargetTemplate struct {
	TargetSpec `json:",inline"`

	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects. May match selectors of replication controllers
	// and services.
	// More info: http://kubernetes.io/docs/user-guide/labels
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}
