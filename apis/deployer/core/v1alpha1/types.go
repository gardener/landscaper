// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsschema "github.com/gardener/landscaper/apis/schema"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OwnerList contains a list of Owners.
type OwnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Owner `json:"items"`
}

// OwnerDefinition defines the Installation resource CRD.
var OwnerDefinition = lsschema.CustomResourceDefinition{
	Names: lsschema.CustomResourceDefinitionNames{
		Plural:   "owners",
		Singular: "owner",
		ShortNames: []string{
			"ow",
		},
		Kind: "Owner",
	},
	Scope:             lsschema.NamespaceScoped,
	Storage:           true,
	Served:            true,
	SubresourceStatus: true,
	AdditionalPrinterColumns: []lsschema.CustomResourceColumnDefinition{
		{
			Name:     "Type",
			Type:     "string",
			JSONPath: ".spec.type",
		},
		{
			Name:     "Deployer",
			Type:     "string",
			JSONPath: ".spec.deployerId",
		},
		{
			Name:     "Accepted",
			Type:     "string",
			JSONPath: ".status.accepted",
		},
	},
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Owner are resources that can hold any kind of json or yaml data.
type Owner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec contains configuration for a deployer ownership.
	Spec   OwnerSpec   `json:"spec"`
	Status OwnerStatus `json:"status"`
}

// OwnerSpec describes the configuration for a deployer ownership.
type OwnerSpec struct {
	// Type describes the type of the deployer.
	Type string `json:"type"`
	// DeployerId describes the unique identity of a deployer.
	DeployerId string `json:"deployerId"`
	// Targets is a list of targets the referenced deployer is responsible for.
	Targets []lsv1alpha1.ObjectReference `json:"targets,omitempty"`
}

// OwnerStatus describes the status for a deployer ownership.
type OwnerStatus struct {
	// Accepted defines if the responsible controller has accepted the owner.
	Accepted bool `json:"accepted"`
	// ObservedGeneration indicates the generation that was last observed by the responsive deployer controller.
	ObservedGeneration int64 `json:"observedGeneration"`
}
