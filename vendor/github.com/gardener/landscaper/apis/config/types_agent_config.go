// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AgentConfiguration contains all configuration for the landscaper agent
type AgentConfiguration struct {
	metav1.TypeMeta `json:",inline"`
	// Name is the name for the agent and environment.
	// This name has to be landscaper globally unique.
	Name string `json:"name"`
	// Namespace is the namespace in the host cluster where the deployers should be installed.
	// Defaults to ls-system
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// OCI defines a oci registry to use for definitions
	// +optional
	OCI *OCIConfiguration `json:"oci,omitempty"`
	// TargetSelectors defines the target selector that is applied to all installed deployers
	// +optional
	TargetSelectors []lsv1alpha1.TargetSelector `json:"targetSelectors,omitempty"`

	// LandscaperNamespace is the namespace in the landscaper cluster where the installations and target for the
	// deployers are stored.
	// Defaults to ls-system
	// +optional
	LandscaperNamespace string `json:"landscaperNamespace,omitempty"`
}
