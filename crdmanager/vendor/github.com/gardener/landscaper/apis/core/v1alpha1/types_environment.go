// SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsschema "github.com/gardener/landscaper/apis/schema"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EnvironmentList contains a list of Environments
type EnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Environment `json:"items"`
}

// EnvironmentDefinition defines the Environment resource CRD.
var EnvironmentDefinition = lsschema.CustomResourceDefinition{
	Names: lsschema.CustomResourceDefinitionNames{
		Plural:   "environments",
		Singular: "environment",
		ShortNames: []string{
			"env",
		},
		Kind: "Environment",
	},
	Scope:   lsschema.ClusterScoped,
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
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Environment defines a environment that is created by a agent.
type Environment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec defines the environment.
	Spec EnvironmentSpec `json:"spec"`
}

// EnvironmentSpec defines the environment configuration.
type EnvironmentSpec struct {
	// HostTarget describes the target that is used for the deployers.
	HostTarget TargetTemplate `json:"hostTarget"`
	// Namespace is the host cluster namespace where the deployers should be installed.
	Namespace string `json:"namespace"`
	// LandscaperClusterRestConfig describes the connection information to connect to the
	// landscaper cluster.
	// This information should be provided by the agent as the access information may differ
	// when calling from different networking zones.
	LandscaperClusterRestConfig ClusterRestConfig `json:"landscaperClusterConfig"`
	// TargetSelector defines the target selector that is applied to all installed deployers
	TargetSelectors []TargetSelector `json:"targetSelectors"`
}

// ClusterRestConfig describes parts of a rest.Config
// that is used to access the
type ClusterRestConfig struct {
	// Host must be a host string, a host:port pair, or a URL to the base of the apiserver.
	// If a URL is given then the (optional) Path of that URL represents a prefix that must
	// be appended to all request URIs used to access the apiserver. This allows a frontend
	// proxy to easily relocate all of the apiserver endpoints.
	Host string `json:"host"`
	// APIPath is a sub-path that points to an API root.
	APIPath string `json:"apiPath"`

	TLSClientConfig `json:",inline"`
}

// TLSClientConfig contains settings to enable transport layer security
type TLSClientConfig struct {
	// Server should be accessed without verifying the TLS certificate. For testing only.
	// +optional
	Insecure bool `json:"insecure"`
	// ServerName is passed to the server for SNI and is used in the client to check server
	// ceritificates against. If ServerName is empty, the hostname used to contact the
	// server is used.
	// +optional
	ServerName string `json:"serverName,omitempty"`

	// CAData holds PEM-encoded bytes (typically read from a root certificates bundle).
	// CAData takes precedence over CAFile
	// +optional
	CAData []byte `json:"caData,omitempty"`

	// NextProtos is a list of supported application level protocols, in order of preference.
	// Used to populate tls.Config.NextProtos.
	// To indicate to the server http/1.1 is preferred over http/2, set to ["http/1.1", "h2"] (though the server is free to ignore that preference).
	// To use only http/1.1, set to ["http/1.1"].
	// +optional
	NextProtos []string `json:"nextProtos,omitempty"`
}
