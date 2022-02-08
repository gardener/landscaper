// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package subinstallations

import "github.com/gardener/landscaper/pkg/landscaper/installations"

// Operation contains all subinstallation operations
type Operation struct {
	*installations.Operation
	Forced bool
}

// New creates a new subinstallation operation
func New(op *installations.Operation) *Operation {
	return &Operation{
		Operation: op,
		Forced:    false,
	}
}
