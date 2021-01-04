// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package terraform

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProviderConfiguration is the terraform deployer configuration that is expected in a DeployItem.
type ProviderConfiguration struct {
	metav1.TypeMeta `json:",inline"`
	// Main is the main terraform configuration.
	Main string `json:"main.tf"`
	// Variables are the terraform variables.
	// +optional
	Variables string `json:"variables.tf,omitempty"`
	// TFVars are the terraform input variables.
	// +optional
	TFVars string `json:"terraform.tfvars,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProviderStatus is the terraform provider specific status.
type ProviderStatus struct {
	metav1.TypeMeta `json:",inline"`
	// Output contains the terraform outputs that are deployed by the terrraform deployer.
	Output json.RawMessage `json:"output"`
}
