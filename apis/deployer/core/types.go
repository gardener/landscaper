// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package core

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lscore "github.com/gardener/landscaper/apis/core"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OwnerList contains a list of Owners.
type OwnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Owner `json:"items"`
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
	Targets []lscore.ObjectReference `json:"targets,omitempty"`
}

// OwnerStatus describes the status for a deployer ownership.
type OwnerStatus struct {
	// Accepted defines if the responsible controller has accepted the owner.
	Accepted bool `json:"accepted"`
	// ObservedGeneration indicates the generation that was last observed by the responsive deployer controller.
	ObservedGeneration int64 `json:"observedGeneration"`
}
