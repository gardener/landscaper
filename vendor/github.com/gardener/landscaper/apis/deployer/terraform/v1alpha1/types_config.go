// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Configuration is the terraform deployer configuration that configures the controller.
type Configuration struct {
	metav1.TypeMeta `json:",inline"`
	// TargetSelector describes all selectors the deployer should depend on.
	TargetSelector []lsv1alpha1.TargetSelector `json:"targetSelector,omitempty"`
	// Terraformer contains the configuration of the Terraformer.
	Terraformer TerraformerSpec `json:"terraformer"`
}

// TerraformerSpec is the configuration for the Terraformer.
type TerraformerSpec struct {
	// Namespace is the namespace where the Terraformer will run and store the resources.
	// The namespace will be defaulted to the default namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// Image is the name of the terraformer container image.
	// The image will be defaulted by the terraform deployer to the configured default.
	// +optional
	Image string `json:"image,omitempty"`
	// LogLevel is the log level of the terraformer.
	// Default to info.
	// +optional
	LogLevel string `json:"logLevel,omitempty"`
}
