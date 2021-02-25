// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProviderConfiguration is the mock deployer configuration that is expected in a DeployItem
type ProviderConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// Phase sets the phase of the DeployItem
	Phase *lsv1alpha1.ExecutionPhase `json:"phase,omitempty"`

	// ProviderStatus sets the provider status to the given value
	ProviderStatus *runtime.RawExtension `json:"providerStatus,omitempty"`

	// Export sets the exported configuration to the given value
	Export *json.RawMessage `json:"export,omitempty"`
}
