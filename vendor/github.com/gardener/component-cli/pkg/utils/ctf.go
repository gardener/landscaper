// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"strings"
)

// CTFComponentArchiveFilename returns the name of the componant archive file in the ctf.
func CTFComponentArchiveFilename(name, version string) string {
	return fmt.Sprintf("%s-%s.tar", strings.ReplaceAll(name, "/", "_"), version)
}
