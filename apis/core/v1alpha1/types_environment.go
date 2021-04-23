// SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EnvironmentList contains a list of Environments
type EnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Environment `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Environment defines a environment that is created by a agent.
// +kubebuilder:resource:path="environments",scope="Cluster",shortName="env",singular="environment"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=Age,type=date
type Environment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec defines the environment.
	Spec EnvironmentSpec `json:"spec"`
}

// EnvironmentSpec defines the environment configuration.
type EnvironmentSpec struct {
	// HostTarget points to the target that is created by the agent with the environment.
	HostTarget ObjectReference `json:"hostTarget"`
	// TargetSelector defines the target selector that is applied to all installed deployers
	TargetSelector TargetSelector `json:"targetSelector"`
}
