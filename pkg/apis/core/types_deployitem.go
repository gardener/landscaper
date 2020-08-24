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
	"k8s.io/apimachinery/pkg/runtime"
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
// +kubebuilder:subresource:status
type DeployItem struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DeployItemSpec `json:"spec"`

	// +optional
	Status DeployItemStatus `json:"status"`
}

// DeployItemSpec contains the definition of a deploy item.
type DeployItemSpec struct {
	// DataType is the type of the deployer that should handle the item.
	Type ExecutionType `json:"type"`

	// BlueprintRef is the resolved reference to the definition.
	// +optional
	BlueprintRef *RemoteBlueprintReference `json:"blueprintRef,omitempty"`

	// RegistryPullSecrets defines a list of registry credentials that are used to
	// pull blueprints and component descriptors from the respective registry.
	// For more info see: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/
	// Note that the type information is used to determine the secret key and the type of the secret.
	// +optional
	RegistryPullSecrets []ObjectReference `json:"registryPullSecrets,omitempty"`

	// ImportReference is the reference to the object containing all imported values.
	// +optional
	ImportReference *ObjectReference `json:"importRef,omitempty"`

	// ProviderConfiguration contains the deployer type specific configuration.
	Configuration runtime.RawExtension `json:"config,omitempty"`
}

// DeployItemStatus contains the status of a deploy item
type DeployItemStatus struct {
	// Phase is the current phase of the DeployItem
	Phase ExecutionPhase `json:"phase,omitempty"`

	// ObservedGeneration is the most recent generation observed for this DeployItem.
	// It corresponds to the DeployItem generation, which is updated on mutation by the landscaper.
	ObservedGeneration int64 `json:"observedGeneration"`

	// Conditions contains the actual condition of a deploy item
	// +optional
	Conditions []Condition `json:"conditions,omitempty"`

	// ProviderStatus contains the provider specific status
	// +optional
	ProviderStatus runtime.RawExtension `json:"providerStatus,omitempty"`

	// ExportReference is the reference to the object that contains the exported values.
	// +optional
	ExportReference *ObjectReference `json:"exportRef,omitempty"`
}
