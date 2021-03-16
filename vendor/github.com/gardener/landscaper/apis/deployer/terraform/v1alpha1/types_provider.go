// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
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
	// EnvVars defines all environment variables available during the terraform execution.
	// Can be used to pass credentials or configuration data.
	// +optional
	EnvVars []EnvVar `json:"envVars,omitempty"`
	// Files defines all files that are mounted to the terraform container.
	// +optional
	Files []FileMount `json:"files,omitempty"`
	// TerraformProviders defines a list of terraform deployers that terraform should be executed with.
	TerraformProviders TerraformProviders `json:"terraformProviders,omitempty"`
}

// EnvVar represents an environment variable present in a Container.
type EnvVar struct {
	// Name of the environment variable. Must be a C_IDENTIFIER.
	Name string `json:"name"`

	// Optional: no more than one of the following may be specified.

	// Variable references $(VAR_NAME) are expanded
	// using the previous defined environment variables in the container and
	// any service environment variables. If a variable cannot be resolved,
	// the reference in the input string will be unchanged. The $(VAR_NAME)
	// syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped
	// references will never be expanded, regardless of whether the variable
	// exists or not.
	// Defaults to "".
	// +optional
	Value string `json:"value,omitempty"`
	// FromTarget defines the source for the environment variable's value from the given target's configuration.
	// Cannot be used if value is not empty.
	// +optional
	FromTarget *FromTarget `json:"fromTarget,omitempty"`
}

// FileMount defines a file that is mounted to a Container.
type FileMount struct {
	// Name of the environment variable.
	Name string `json:"name"`
	// Value defines a raw base64 encoded value.
	// Defaults to "".
	// +optional
	Value string `json:"value,omitempty"`
	// FromTarget defines the source for the environment variable's value from the given target's configuration.
	// Cannot be used if value is not empty.
	// +optional
	FromTarget *FromTarget `json:"fromTarget,omitempty"`
}

// FromTarget defines the value from a target.
// The value can be specified from the target using a jsonpath.
type FromTarget struct {
	JSONPath string `json:"jsonPath"`
}

// TerraformProviders defines a list of terraform deployers.
type TerraformProviders []TerraformProvider

// TerraformProvider defines a remote terraform provider that is used in the execution.
type TerraformProvider struct {
	// Name of the terraform provider.
	// e.g. "aws"
	Name string
	// Version of the terraform provider
	Version string
	// Inline defines a inline terraform provider that is already available with the deployer.
	// +optional
	Inline string `json:"inline,omitempty"`
	// URL defines a remote url where the provider should downloaded from.
	// +optional
	URL string `json:"url,omitempty"`
	// FromResource defines a reference to a remote terraform provider through a Component-Descriptor.
	// +optional
	FromResource *RemoteTerraformReference `json:"fromResource,omitempty"`
}

// RemoteTerraformReference defines a reference to a remote terraform provider through a Component-Descriptor
type RemoteTerraformReference struct {
	lsv1alpha1.ComponentDescriptorDefinition `json:",inline"`
	// ResourceName is the name of the terraform provider as defined by a component descriptor.
	ResourceName string `json:"resourceName"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProviderStatus is the terraform provider specific status.
type ProviderStatus struct {
	metav1.TypeMeta `json:",inline"`
	// Output contains the terraform outputs that are deployed by the terrraform deployer.
	Output json.RawMessage `json:"output"`
}
