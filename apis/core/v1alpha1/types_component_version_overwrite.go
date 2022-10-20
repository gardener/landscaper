// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsschema "github.com/gardener/landscaper/apis/schema"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ComponentVersionOverwritesList contains a list of ComponentVersionOverwrites
type ComponentVersionOverwritesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ComponentVersionOverwrites `json:"items"`
}

// ComponentVersionOverwritesDefinition defines the ComponentVersionOverwrites resource CRD.
var ComponentVersionOverwritesDefinition = lsschema.CustomResourceDefinition{
	Names: lsschema.CustomResourceDefinitionNames{
		Plural:   "componentversionoverwrites",
		Singular: "componentversionoverwrite",
		ShortNames: []string{
			"compveroverwrite",
			"cvo",
		},
		Kind: "ComponentVersionOverwrites",
	},
	Scope:   lsschema.NamespaceScoped,
	Storage: true,
	Served:  true,
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ComponentVersionOverwrites contain overwrites for specific (versions of) components.
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
