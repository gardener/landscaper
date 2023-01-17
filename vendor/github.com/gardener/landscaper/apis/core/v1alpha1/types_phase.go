// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

// BasePhase is a common super type for all phases
type BasePhase string

func (p BasePhase) Phase() BasePhase {
	return p
}

// Phase is a common interface for the different kinds of phases
type Phase interface {
	// Phase returns the phase as BasePhase
	Phase() BasePhase
}

const (
	PhaseInit            BasePhase = "Init"
	PhaseCleanupOrphaned BasePhase = "CleanupOrphaned"
	PhaseObjectsCreated  BasePhase = "ObjectsCreated"
	PhaseProgressing     BasePhase = "Progressing"
	PhaseCompleting      BasePhase = "Completing"
	PhaseSucceeded       BasePhase = "Succeeded"
	PhaseFailed          BasePhase = "Failed"

	PhaseInitDelete    BasePhase = "InitDelete"
	PhaseTriggerDelete BasePhase = "TriggerDelete"
	PhaseDeleting      BasePhase = "Deleting"
	PhaseDeleteFailed  BasePhase = "DeleteFailed"
)
