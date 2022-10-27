// SPDX-FileCopyrightText: 2021 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsschema "github.com/gardener/landscaper/apis/schema"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TargetSyncList contains a list of TargetSync objects
type TargetSyncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TargetSync `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// The TargetSync is created targets from secrets.
type TargetSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec contains the specification
	Spec TargetSyncSpec `json:"spec"`

	// Status contains the status
	// +optional
	Status TargetSyncStatus `json:"status"`
}

// TargetSyncSpec contains the specification for a TargetSync.
type TargetSyncSpec struct {
	// SecretRef references the secret that contains the kubeconfig to the name.
	SecretRef SecretReference `json:"secretRef"`

	// SourceNamespace describes the namespace from where the secrets should be synced
	SourceNamespace string `json:"sourceNamespace"`

	// SecretNameExpression defines the names of the secrets which should be synced via a regular expression according
	// to https://github.com/google/re2/wiki/Syntax
	// if not set all secrets are synced
	// +optional
	SecretNameExpression string `json:"secretNameExpression"`
}

// TargetSyncStatus contains the status of a TargetSync.
type TargetSyncStatus struct {
	// ObservedGeneration is the most recent generation observed.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration"`

	// LastErrors describe the last errors
	// +optional
	LastErrors []string `json:"lastErrors,omitempty"`
}

var TargetSyncDefinition = lsschema.CustomResourceDefinition{
	Names: lsschema.CustomResourceDefinitionNames{
		Plural:   "targetsyncs",
		Singular: "targetsync",
		ShortNames: []string{
			"tgs",
		},
		Kind: "TargetSync",
	},
	Scope:                    lsschema.NamespaceScoped,
	Storage:                  true,
	Served:                   true,
	SubresourceStatus:        true,
	AdditionalPrinterColumns: []lsschema.CustomResourceColumnDefinition{},
}
