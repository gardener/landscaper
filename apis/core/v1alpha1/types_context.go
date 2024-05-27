// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DefaultContextName is the name of the default context that is replicated in all namespaces.
const DefaultContextName = "default"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ContextList contains a list of Contexts
type ContextList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Context `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:shortName=ctx
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Context is a resource that contains shared information of installations.
// This includes information about the repository context like the context itself or secrets to access the oci artifacts.
// But it can also contain deployer specific config.
type Context struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	ContextConfiguration `json:",inline"`
}

type ContextConfiguration struct {
	// RepositoryContext defines the context of the component repository to resolve blueprints.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +optional
	RepositoryContext *cdv2.UnstructuredTypedObject `json:"repositoryContext,omitempty"`
	// UseOCM defines whether OCM is used to process installations that reference this context.
	// +optional
	UseOCM bool `json:"useOCM,omitempty"`
	// OCMConfig references a k8s config map object that contains the ocm configuration data in the format of an
	// ocm configfile.
	// For more info see: https://github.com/open-component-model/ocm/blob/main/docs/reference/ocm_configfile.md
	// +optional
	OCMConfig *corev1.LocalObjectReference `json:"ocmConfig"`
	// RegistryPullSecrets defines a list of registry credentials that are used to
	// pull blueprints, component descriptors and jsonschemas from the respective registry.
	// For more info see: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/
	// Note that the type information is used to determine the secret key and the type of the secret.
	// +optional
	RegistryPullSecrets []corev1.LocalObjectReference `json:"registryPullSecrets,omitempty"`
	// Configurations contains arbitrary configuration information for dedicated purposes given by a string key.
	// The key should use a dns-like syntax to express the purpose and avoid conflicts.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	// +optional
	Configurations map[string]AnyJSON `json:"configurations,omitempty"`
	// ComponentVersionOverwritesReference is a reference to a ComponentVersionOverwrites object
	// The overwrites object has to be in the same namespace as the context.
	// If the string is empty, no overwrites will be used.
	// +optional
	ComponentVersionOverwritesReference string `json:"componentVersionOverwrites"`

	// VerificationSignatures maps a signature name to the trusted verification information
	// +optional
	VerificationSignatures map[string]VerificationSignature `json:"verificationSignatures,omitempty"`
}

// VerificationSignatures contains the trusted verification information
type VerificationSignature struct {
	// PublicKeySecretReference contains a secret reference to a public key in PEM format that is used to verify the component signature
	PublicKeySecretReference *SecretReference `json:"publicKeySecretReference,omitempty"`
	// CaCertificateSecretReference contains a secret reference to one or more certificates in PEM format that are used to verify the compnent signature
	CaCertificateSecretReference *SecretReference `json:"caCertificateSecretReference,omitempty"`
}
