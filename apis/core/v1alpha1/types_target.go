// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:shortName=tgt;tg
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Context",type=string,JSONPath=`.metadata.labels['data\.landscaper\.gardener\.cloud\/context']`
// +kubebuilder:printcolumn:name="Key",type=string,JSONPath=`.metadata.labels['data\.landscaper\.gardener\.cloud\/key']`
// +kubebuilder:printcolumn:name="Idx",type=string,JSONPath=`.metadata.labels['data\.landscaper\.gardener\.cloud\/index']`
// +kubebuilder:printcolumn:name="TMKey",type=string,JSONPath=`.metadata.labels['data\.landscaper\.gardener\.cloud\/targetmapkey']`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Target defines a specific data object that defines target environment.
// Every deploy item can have a target which is used by the deployer to install the specific application.
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
	// Exactly one of the fields Configuration and SecretRef must be set
	// +kubebuilder:validation:Schemaless
	// +optional
	Configuration *AnyJSON `json:"config,omitempty"`

	// Reference to a secret containing the target type specific configuration.
	// Exactly one of the fields Configuration and SecretRef must be set
	// +optional
	SecretRef *LocalSecretReference `json:"secretRef,omitempty"`
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

// ResolvedTarget is a helper struct to store a target together with the content of its resolved secret reference.
type ResolvedTarget struct {
	// Target contains the original target.
	*Target `json:"target"`

	// Content contains the content of the target.
	// If the target has a secret reference, this field should be filled by a TargetResolver.
	// Otherwise, the inline configuration of the target is put here.
	Content string `json:"content"`
}

// NewResolvedTarget is a constructor for ResolvedTarget.
// It puts the target's inline configuration into the Content field, if the target doesn't contain a secret reference.
func NewResolvedTarget(target *Target) *ResolvedTarget {
	res := &ResolvedTarget{
		Target: target,
	}
	if target.Spec.SecretRef == nil && target.Spec.Configuration != nil {
		res.Content = string(target.Spec.Configuration.RawMessage)
	}
	return res
}
