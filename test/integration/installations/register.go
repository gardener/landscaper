// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"time"

	"github.com/gardener/landscaper/test/framework"
)

var (
	resyncTime  = 1 * time.Second
	timeoutTime = 30 * time.Second
)

// RegisterTests registers all tests of this package
func RegisterTests(f *framework.Framework) {
	ImportExportTests(f)
}
