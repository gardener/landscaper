// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package installations

import (
	"fmt"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// ImportStatus is the internal representation of all import status of a installation.
type ImportStatus struct {
	From map[string]*lsv1alpha1.ImportState
	To   map[string]*lsv1alpha1.ImportState
}

func (s *ImportStatus) set(status lsv1alpha1.ImportState) {
	s.From[status.From] = &status
	s.To[status.To] = &status
}

// Updates the internal import states
func (s *ImportStatus) Update(state lsv1alpha1.ImportState) {
	s.set(state)
}

// GetStates returns the import states of the installation.
func (s *ImportStatus) GetStates() []lsv1alpha1.ImportState {
	states := make([]lsv1alpha1.ImportState, 0)
	for _, state := range s.To {
		states = append(states, *state)
	}
	return states
}

// GetFrom returns the component state for the given From key.
func (s *ImportStatus) GetFrom(key string) (lsv1alpha1.ImportState, error) {
	state, ok := s.From[key]
	if !ok {
		return lsv1alpha1.ImportState{}, fmt.Errorf("import state with from key %s not found", key)
	}
	return *state, nil
}

// GetFrom returns the component state for the given To key.
func (s *ImportStatus) GetTo(key string) (lsv1alpha1.ImportState, error) {
	state, ok := s.To[key]
	if !ok {
		return lsv1alpha1.ImportState{}, fmt.Errorf("import state with to key %s not found", key)
	}
	return *state, nil
}
