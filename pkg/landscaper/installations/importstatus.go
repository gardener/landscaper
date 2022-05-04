// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"fmt"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// ImportStatus is the internal representation of all import status of a installation.
type ImportStatus struct {
	Data                map[string]*lsv1alpha1.ImportStatus
	Target              map[string]*lsv1alpha1.ImportStatus
	ComponentDescriptor map[string]*lsv1alpha1.ImportStatus
}

func (s *ImportStatus) set(status lsv1alpha1.ImportStatus) {
	if status.Type == lsv1alpha1.DataImportStatusType {
		s.Data[status.Name] = &status
	}
	if status.Type == lsv1alpha1.TargetImportStatusType || status.Type == lsv1alpha1.TargetListImportStatusType {
		s.Target[status.Name] = &status
	}
	if status.Type == lsv1alpha1.CDImportStatusType || status.Type == lsv1alpha1.CDListImportStatusType {
		s.ComponentDescriptor[status.Name] = &status
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
	for _, state := range s.ComponentDescriptor {
		states = append(states, *state)
	}
	return states
}

// GetData returns the import data status for the given key.
func (s *ImportStatus) GetData(name string) (lsv1alpha1.ImportStatus, error) {
	state, ok := s.Data[name]
	if !ok {
		return lsv1alpha1.ImportStatus{}, fmt.Errorf("import state %s not found", name)
	}
	return *state, nil
}

// GetTarget returns the import target state for the given key.
func (s *ImportStatus) GetTarget(name string) (lsv1alpha1.ImportStatus, error) {
	state, ok := s.Target[name]
	if !ok {
		return lsv1alpha1.ImportStatus{}, fmt.Errorf("import state %s not found", name)
	}
	return *state, nil
}

// GetComponentDescriptor returns the import target state for the given key.
func (s *ImportStatus) GetComponentDescriptor(name string) (lsv1alpha1.ImportStatus, error) {
	state, ok := s.ComponentDescriptor[name]
	if !ok {
		return lsv1alpha1.ImportStatus{}, fmt.Errorf("import state %s not found", name)
	}
	return *state, nil
}
