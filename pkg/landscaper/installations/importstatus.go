// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// ImportStatus is the internal representation of all import status of an installation.
type ImportStatus struct {
	Data   map[string]*lsv1alpha1.ImportStatus
	Target map[string]*lsv1alpha1.ImportStatus
}

func (s *ImportStatus) set(status lsv1alpha1.ImportStatus) {
	if status.Type == lsv1alpha1.DataImportStatusType {
		s.Data[status.Name] = &status
	}
	if status.Type == lsv1alpha1.TargetImportStatusType || status.Type == lsv1alpha1.TargetListImportStatusType {
		s.Target[status.Name] = &status
	}
}

// Update the internal import states
func (s *ImportStatus) Update(state lsv1alpha1.ImportStatus) {
	s.set(state)
}

// GetStatus returns the import states of the installation.
func (s *ImportStatus) GetStatus() []lsv1alpha1.ImportStatus {
	states := make([]lsv1alpha1.ImportStatus, 0)
	for _, state := range s.Data {
		states = append(states, *state)
	}
	for _, state := range s.Target {
		states = append(states, *state)
	}

	return states
}
