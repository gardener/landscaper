// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imports

import (
	"github.com/gardener/landscaper/pkg/landscaper/installations"
)

// Validators is a struct that contains everything to
// validate if all imports of a installation are satisfied.
type Validator struct {
	*installations.Operation

	parent   *installations.Installation
	siblings []installations.InstallationBaseInterface
}

// Constructor is a struct that contains all values
// that are needed to load all imported data and
// generate the one imported config
type Constructor struct {
	*installations.Operation
	validator *Validator

	parent   *installations.Installation
	siblings []installations.InstallationBaseInterface
}
