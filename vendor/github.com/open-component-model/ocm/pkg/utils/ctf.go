// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"strings"
)

// CTFComponentArchiveFilename returns the name of the component archive file in the ctf.
func CTFComponentArchiveFilename(name, version string) string {
	return fmt.Sprintf("%s-%s.tar", strings.ReplaceAll(name, "/", "_"), version)
}
