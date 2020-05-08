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

package component

import (
	corev1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// Component is the internal representation of a component
type Component struct {
	Info *corev1alpha1.Component

	importStates map[string]*corev1alpha1.ImportState
}

func New(component *corev1alpha1.Component) (*Component, error) {

	c := &Component{
		Info:         component,
		importStates: make(map[string]*corev1alpha1.ImportState, len(component.Status.Imports)),
	}

	for _, state := range component.Status.Imports {
		c.importStates[state.From] = state.DeepCopy()
	}

	return c, nil
}

func (c *Component) GetImportStatus(from string) (*corev1alpha1.ImportState, bool) {
	s, ok := c.importStates[from]
	return s, ok
}
