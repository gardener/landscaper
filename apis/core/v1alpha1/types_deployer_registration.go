// SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsschema "github.com/gardener/landscaper/apis/schema"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DeployerRegistrationList contains a list of DeployerRegistration
type DeployerRegistrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeployerRegistration `json:"items"`
}

// DeployerRegistrationDefinition defines the DeployerRegistration resource CRD.
var DeployerRegistrationDefinition = lsschema.CustomResourceDefinition{
	Names: lsschema.CustomResourceDefinitionNames{
		Plural:   "deployerregistrations",
		Singular: "deployerregistration",
		ShortNames: []string{
			"deployreg",
			"deployerreg",
			"dreg",
		},
		Kind: "DeployerRegistration",
	},
	Scope:             lsschema.ClusterScoped,
	Storage:           true,
	Served:            true,
	SubresourceStatus: true,
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

// DeployerRegistration defines a installation template for a deployer.
type DeployerRegistration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec defines the deployer registration configuration.
	Spec DeployerRegistrationSpec `json:"spec"`
	// Status contains the current status of the deployer registration.
	// +optional
	Status DeployerRegistrationStatus `json:"status"`
}

// DeployerRegistrationSpec defines the configuration of a deployer registration
type DeployerRegistrationSpec struct {
	// DeployItemTypes defines the types of deploy items that are handled by the deployer.
	DeployItemTypes []DeployItemType `json:"types"`
	// InstallationTemplate defines the installation template for installing a deployer.´
	InstallationTemplate DeployerInstallationTemplate `json:"installationTemplate"`
}

type DeployerInstallationTemplate struct {
	//ComponentDescriptor is a reference to the installation's component descriptor
	// +optional
	ComponentDescriptor *ComponentDescriptorDefinition `json:"componentDescriptor,omitempty"`
	// Blueprint is the resolved reference to the definition.
	Blueprint BlueprintDefinition `json:"blueprint"`
	// Imports define the imported data objects and targets.
	// +optional
	Imports InstallationImports `json:"imports,omitempty"`
	// ImportDataMappings contains a template for restructuring imports.
	// It is expected to contain a key for every blueprint-defined data import.
	// Missing keys will be defaulted to their respective data import.
	// Example: namespace: (( installation.imports.namespace ))
	// +optional
	ImportDataMappings map[string]AnyJSON `json:"importDataMappings,omitempty"`
}

// DeployerRegistrationStatus contains the current status of a deployer registration.
type DeployerRegistrationStatus struct {
	// LastError describes the last error that occurred.
	LastError *Error `json:"lastError,omitempty"`
}
