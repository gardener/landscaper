// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package core

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LsHealthCheckList contains a list of LsHealthChecks
type LsHealthCheckList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LsHealthCheck `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LsHealthCheck is a resource containing information about problems with the landscaper installation
type LsHealthCheck struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Status defines the overall status of the landscaper and its standard deployers.
	Status LsHealthCheckStatus `json:"status"`

	// LastUpdateTime contains last time the check was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime"`

	// Description contains description of the problem(s)
	Description string `json:"description"`
}

type LsHealthCheckStatus string
