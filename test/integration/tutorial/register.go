// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package tutorial

import (
	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/test/framework"
)

// RegisterTests registers all tests of the package
func RegisterTests(f *framework.Framework) {
	if utils.IsNewReconcile() {
		NginxIngressTestForNewReconcile(f)
		SimpleImportForNewReconcile(f)
		AggregatedBlueprintForNewReconcile(f)
		ExternalJSONSchemaTestForNewReconcile(f)
	} else {
		NginxIngressTest(f)
		SimpleImport(f)
		AggregatedBlueprint(f)
		ExternalJSONSchemaTest(f)
	}
}
