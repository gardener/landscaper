// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployers

import (
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/integration/deployers/container"
)

// RegisterTests registers all tests of this package
func RegisterTests(f *framework.Framework) {
	//ManifestDeployerTestsForNewReconcile(f)
	//helmcharts.RegisterTests(f)
	container.ContainerTests(f)
	//blueprints.RegisterTests(f)
	//management.RegisterTests(f)
	//helmdeployer.RegisterTests(f)
}
