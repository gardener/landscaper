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
	"k8s.io/apimachinery/pkg/runtime"
)

// ExecutionManagedByLabel is the label of a deploy item that contains the name of the managed execution.
// This label is used by the extension controller to identify its managed deploy items
// todo: add conversion
const ExecutionManagedByLabel = "execution.landscaper.gardener.cloud/managed-by"

// ExecutionManagedNameAnnotation is the unique identifier of the deploy items managed by a execution.
// It corresponds to the execution item name.
// todo: add conversion
const ExecutionManagedNameAnnotation = "execution.landscaper.gardener.cloud/name"

// ExecutionType defines the type of the execution
type ExecutionType string

// ReconcileDeployItemsCondition is the Conditions type to indicate the deploy items status.
const ReconcileDeployItemsCondition ConditionType = "ReconcileDeployItems"

type ExecutionPhase string

const (
	ExecutionPhaseInit        = ExecutionPhase(ComponentPhaseInit)
	ExecutionPhaseProgressing = ExecutionPhase(ComponentPhaseProgressing)
	ExecutionPhaseDeleting    = ExecutionPhase(ComponentPhaseDeleting)
	ExecutionPhaseSucceeded   = ExecutionPhase(ComponentPhaseSucceeded)
	ExecutionPhaseFailed      = ExecutionPhase(ComponentPhaseFailed)
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
// +kubebuilder:resource:path="executions",scope="Namespaced",shortName="exec",singular="execution"
// +kubebuilder:printcolumn:JSONPath=".status.phase",name=Phase,type=string
// +kubebuilder:printcolumn:JSONPath=".status.exportRef.name",name=ExportRef,type=string
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=Age,type=date
// +kubebuilder:subresource:status
type Execution struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec defines a execution and its items
	Spec ExecutionSpec `json:"spec"`
	// Status contains the current status of the execution.
	// +optional
	Status ExecutionStatus `json:"status"`
}

// ExecutionSpec defines a execution plan.
type ExecutionSpec struct {
	// BlueprintRef is the resolved reference to the definition.
	// +optional
	BlueprintRef *RemoteBlueprintReference `json:"blueprintRef"`

	// RegistryPullSecrets defines a list of registry credentials that are used to
	// pull blueprints and component descriptors from the respective registry.
	// For more info see: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/
	// Note that the type information is used to determine the secret key and the type of the secret.
	// +optional
	RegistryPullSecrets []ObjectReference `json:"registryPullSecrets"`

	// ImportReference is the reference to the object containing all imported values.
	ImportReference *ObjectReference `json:"importRef,omitempty"`

	// Executions defines all execution items that need to be scheduled.
	Executions []DeployItemTemplate `json:"executions"`
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
	// only used for operation purpose.
	// +optional
	ExportReference *ObjectReference `json:"exportRef,omitempty"`

	// DeployItemReferences contain the state of all deploy items.
	// The observed generation is here the generation of the Execution not the DeployItem.
	DeployItemReferences []VersionedNamedObjectReference `json:"deployItemRefs,omitempty"`
}

// DeployItemTemplate defines a execution element that is translated into a deploy item.
type DeployItemTemplate struct {
	// Name is the unique name of the execution.
	Name string `json:"name"`

	// DataType is the DeployItem type of the execution.
	Type ExecutionType `json:"type"`

	// ProviderConfiguration contains the type specific configuration for the execution.
	Configuration runtime.RawExtension `json:"config"`
}
