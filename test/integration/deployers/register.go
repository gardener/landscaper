// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployers

import (
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/integration/deployers/blueprints"
)

// RegisterTests registers all tests of this package
func RegisterTests(f *framework.Framework) {
	//ContainerDeployerTests(f)
	//ManifestDeployerTests(f)
	//helmcharts.RegisterTests(f)
	blueprints.RegisterTests(f)
	//management.RegisterTests(f)
	//continuousreconcile.RegisterTests(f)
}
