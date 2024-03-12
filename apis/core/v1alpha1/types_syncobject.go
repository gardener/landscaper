// SPDX-FileCopyrightText: 2021 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SyncObjectList contains a list of SyncObject objects
type SyncObjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SyncObject `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:shortName=so
// +kubebuilder:printcolumn:name="PodName",type=string,JSONPath=`.spec.podName`
// +kubebuilder:printcolumn:name="Kind",type=string,JSONPath=`.spec.kind`
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
// +kubebuilder:subresource:status

// The SyncObject helps to sync access to deploy items.
type SyncObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec contains the specification
	Spec SyncObjectSpec `json:"spec"`

	// Status contains the status
	// +optional
	Status SyncObjectStatus `json:"status"`
}

// SyncObjectSpec contains the specification.
type SyncObjectSpec struct {
	// PodName describes the name of the pod of the responsible deployer
	PodName string `json:"podName"`

	// Kind describes the kind of object that is being locked by this SyncObject
	Kind string `json:"kind"`

	// Name is the name of the object that is being locked by this SyncObject
	Name string `json:"name"`

	// LastUpdateTime contains last time the object was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime"`

	// Prefix is the prefix of the name of the object.
	Prefix string `json:"prefix"`
}

// SyncObjectStatus contains the status.
type SyncObjectStatus struct {
}
