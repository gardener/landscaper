// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package core

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
