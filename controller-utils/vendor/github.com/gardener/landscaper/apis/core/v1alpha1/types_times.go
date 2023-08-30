// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type TransitionTimes struct {

	// TriggerTime is the time when the jobID is set.
	// +optional
	TriggerTime *metav1.Time `json:"triggerTime,omitempty"`

	// InitTime is the time when the Init phase starts.
	// +optional
	InitTime *metav1.Time `json:"initTime,omitempty"`

	// WaitTime is the time when the work is done.
	// +optional
	WaitTime *metav1.Time `json:"waitTime,omitempty"`

	// FinishedTime is the time when the finished phase is set.
	// +optional
	FinishedTime *metav1.Time `json:"finishedTime,omitempty"`
}
