// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package inline

import (
	"github.com/gardener/landscaper/test/framework"
)

// RegisterTests registers all tests of this package
func RegisterTests(f *framework.Framework) {
	InlineBlueprintTests(f)
	InlineTemplateTests(f)
	ContextTests(f)
}
