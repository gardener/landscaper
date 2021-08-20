// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	cr "github.com/gardener/landscaper/apis/deployer/utils/continuousreconcile"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProviderConfiguration is the mock deployer configuration that is expected in a DeployItem
type ProviderConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// Phase sets the phase of the DeployItem
	Phase *lsv1alpha1.ExecutionPhase `json:"phase,omitempty"`

	// InitialPhase sets the phase of the DeployItem, but only if it is empty or "Init"
	// Additionally, setting it will suppress the DeployItem phase being set to "Succeeded" after successful reconciliation
	InitialPhase *lsv1alpha1.ExecutionPhase `json:"initialPhase,omitempty"`

	// ProviderStatus sets the provider status to the given value
	ProviderStatus *runtime.RawExtension `json:"providerStatus,omitempty"`

	// Export sets the exported configuration to the given value
	Export *json.RawMessage `json:"export,omitempty"`

	// ContinuousReconcile contains the schedule for continuous reconciliation.
	// +optional
	ContinuousReconcile *cr.ContinuousReconcileSpec `json:"continuousReconcile,omitempty"`
}
