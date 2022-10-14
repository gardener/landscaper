// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package core

import (
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ComponentVersionOverwritesList contains a list of ComponentVersionOverwrites
type ComponentVersionOverwritesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ComponentVersionOverwrites `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ComponentVersionOverwrites contain overwrites for specific (versions of) components.
// +kubebuilder:resource:path="componentversionoverwrites",scope="Cluster",shortName={"compveroverwrite","cvo","overwrite"},singular="componentversionoverwrite"
type ComponentVersionOverwrites struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Overwrites defines a list of component overwrites
	Overwrites ComponentVersionOverwriteList `json:"overwrites,omitempty"`
}

// ComponentVersionOverwriteList is a list of component overwrites.
type ComponentVersionOverwriteList []ComponentVersionOverwrite

// ComponentVersionOverwrite defines an overwrite for a specific component and/or version of a component.
type ComponentVersionOverwrite struct {
	// Source defines the component that should be replaced.
	Source ComponentVersionOverwriteReference `json:"source"`
	// Substitution defines the replacement target for the component or version.
	Substitution ComponentVersionOverwriteReference `json:"substitution"`
}

// ComponentVersionOverwriteReference defines a component reference by
type ComponentVersionOverwriteReference struct {
	// RepositoryContext defines the context of the component repository to resolve blueprints.
	// +optional
	RepositoryContext *cdv2.UnstructuredTypedObject `json:"repositoryContext,omitempty"`
	// ComponentName defines the unique of the component containing the resource.
	// +optional
	ComponentName string `json:"componentName"`
	// Version defines the version of the component.
	// +optional
	Version string `json:"version"`
}
