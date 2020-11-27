// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package terraform

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProviderConfiguration is the terraform deployer configuration that is expected in a DeployItem
type ProviderConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// Kubeconfig is the base64 encoded kubeconfig file.
	// By default the configured target is used to deploy the resources.
	// +optional
	Kubeconfig string `json:"kubeconfig"`

	// Namespace is the namespace where the Terraformer will store the resources
	// and where the secret for the terraform provider must exist.
	// The namespace will be defaulted to the default namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// TerraformerImage is the name of the terraformer container image.
	// The image will be defaulted by the terraform deployer to the configured default.
	// +optional
	TerraformerImage string `json:"terraformerImage,omitempty"`

	// Main is the main terraform configuration.
	Main string `json:"main.tf"`

	// Variables are the terraform variables.
	// +optional
	Variables string `json:"variables.tf,omitempty"`

	// TFVars are the terraform input variables.
	// +optional
	TFVars string `json:"terraform.tfvars,omitempty"`

	// EnvSecrets are the names of the secrets containing environment variables.
	// +optional
	EnvSecrets []string `json:"envSecrets,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProviderStatus is the terraform provider specific status
type ProviderStatus struct {
	metav1.TypeMeta `json:",inline"`

	// Output contains the terraform outputs that are deployed by the terrraform deployer.
	Output json.RawMessage `json:"output"`
}
