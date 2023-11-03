// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsschema "github.com/gardener/landscaper/apis/schema"
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

// ContextDefinition defines the Context resource CRD.
var ContextDefinition = lsschema.CustomResourceDefinition{
	Names: lsschema.CustomResourceDefinitionNames{
		Plural:   "contexts",
		Singular: "context",
		ShortNames: []string{
			"ctx",
		},
		Kind: "Context",
	},
	Scope:   lsschema.NamespaceScoped,
	Storage: true,
	Served:  true,
	AdditionalPrinterColumns: []lsschema.CustomResourceColumnDefinition{
		{
			Name:     "Age",
			Type:     "date",
			JSONPath: ".metadata.creationTimestamp",
		},
	},
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

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
	// +optional
	RepositoryContext *cdv2.UnstructuredTypedObject `json:"repositoryContext,omitempty"`
	// UseOCM defines whether OCM is used to process installations that reference this context.
	// +optional
	UseOCM bool `json:"useOCM,omitempty"`
	// RegistryPullSecrets defines a list of registry credentials that are used to
	// pull blueprints, component descriptors and jsonschemas from the respective registry.
	// For more info see: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/
	// Note that the type information is used to determine the secret key and the type of the secret.
	// +optional
	RegistryPullSecrets []corev1.LocalObjectReference `json:"registryPullSecrets,omitempty"`
	// Configurations contains arbitrary configuration information for dedicated purposes given by a string key.
	// The key should use a dns-like syntax to express the purpose and avoid conflicts.
	// +optional
	Configurations map[string]AnyJSON `json:"configurations,omitempty"`
	// ComponentVersionOverwritesReference is a reference to a ComponentVersionOverwrites object
	// The overwrites object has to be in the same namespace as the context.
	// If the string is empty, no overwrites will be used.
	// +optional
	ComponentVersionOverwritesReference string `json:"componentVersionOverwrites"`
}
