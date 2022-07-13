// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployers

import (
	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/integration/deployers/blueprints"
	"github.com/gardener/landscaper/test/integration/deployers/continuousreconcile"
	"github.com/gardener/landscaper/test/integration/deployers/helmcharts"
	"github.com/gardener/landscaper/test/integration/deployers/management"
)

// RegisterTests registers all tests of this package
func RegisterTests(f *framework.Framework) {
	if utils.IsNewReconcile() {
		ContainerDeployerTestsForNewReconcile(f)
		ManifestDeployerTestsForNewReconcile(f)
		helmcharts.RegisterTests(f)
		blueprints.RegisterTests(f)
		management.RegisterTests(f)
	} else {
		ContainerDeployerTests(f)
		ManifestDeployerTests(f)
		helmcharts.RegisterTests(f)
		blueprints.RegisterTests(f)
		management.RegisterTests(f)
		continuousreconcile.RegisterTests(f)
	}
}
