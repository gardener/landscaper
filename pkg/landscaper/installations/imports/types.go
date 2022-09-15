// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imports

import (
	"github.com/gardener/landscaper/pkg/landscaper/installations"
)

// Constructor is a struct that contains all values
// that are needed to load all imported data and
// generate the one imported config
type Constructor struct {
	*installations.Operation
	siblings []*installations.InstallationAndImports
}
