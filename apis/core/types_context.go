// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package core

import (
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ContextList contains a list of Contexts
type ContextList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Context `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Context is a resource that contains shared information of installations.
// This includes information about the repository context like the context itself or secrets to access the oci artifacts.
// But it can also contain deployer specific config.
type Context struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// RepositoryContext defines the context of the component repository to resolve blueprints.
	// +optional
	RepositoryContext *cdv2.UnstructuredTypedObject `json:"repositoryContext,omitempty"`
	// RegistryPullSecrets defines a list of registry credentials that are used to
	// pull blueprints, component descriptors and jsonschemas from the respective registry.
	// For more info see: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/
	// Note that the type information is used to determine the secret key and the type of the secret.
	// +optional
	RegistryPullSecrets []corev1.LocalObjectReference `json:"registryPullSecrets,omitempty"`
}
