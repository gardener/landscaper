// SPDX-FileCopyrightText: 2021 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package core

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CriticalProblemsList contains a list of critical landscaper problems objects
type CriticalProblemsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CriticalProblems `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CriticalProblems contains a list of critical landscaper problems.
type CriticalProblems struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec contains the specification
	Spec CriticalProblemsSpec `json:"spec"`

	// Status contains the status
	// +optional
	Status CriticalProblemsStatus `json:"status"`
}

// CriticalProblemsSpec contains the specification for a CriticalProblems object.
type CriticalProblemsSpec struct {
	CriticalProblems []CriticalProblem `json:"criticalProblem,omitempty"`
}

// CriticalProblemsStatus contains the status of a CriticalProblems object.
type CriticalProblemsStatus struct {
}

// CriticalProblem contains information about one critical problem.
type CriticalProblem struct {
	// PodName contains the name of the pod where the problem occurred
	PodName string `json:"podName,omitempty"`
	// CreationTime contains the timestamp when the problem occured
	CreationTime metav1.Time `json:"creationTime,omitempty"`
	// Description contains an error description
	Description string `json:"description,omitempty"`
}
