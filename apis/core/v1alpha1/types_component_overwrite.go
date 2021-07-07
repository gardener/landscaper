// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsschema "github.com/gardener/landscaper/apis/schema"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ComponentOverwritesList contains a list of ComponentOverwrites
type ComponentOverwritesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ComponentOverwrites `json:"items"`
}

// ComponentOverwritesDefinition defines the ComponentOverwrites resource CRD.
var ComponentOverwritesDefinition = lsschema.CustomResourceDefinition{
	Names: lsschema.CustomResourceDefinitionNames{
		Plural:   "componentoverwrites",
		Singular: "componentoverwrites",
		ShortNames: []string{
			"compoverwrite",
			"co",
			"overwrite",
		},
		Kind: "ComponentOverwrites",
	},
	Scope:   lsschema.ClusterScoped,
	Storage: true,
	Served:  true,
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ComponentOverwrites are resources that can hold any kind json or yaml data.
type ComponentOverwrites struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Overwrites defines a list of component overwrites
	Overwrites ComponentOverwriteList `json:"overwrites,omitempty"`
}

// ComponentOverwriteList is a list of component overwrites.
type ComponentOverwriteList []ComponentOverwrite

// ComponentOverwrite defines an overwrite for a specific component and/or version of a component.
type ComponentOverwrite struct {
	// Component defines the component that should be replaced.
	// The version is optional and will default to all found versions
	Component ComponentOverwriteReference `json:"component"`
	// Target defines the replacement target for the component or version.
	Target ComponentOverwriteReference `json:"target"`
}

// ComponentOverwriteReference defines a component reference by
type ComponentOverwriteReference struct {
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
