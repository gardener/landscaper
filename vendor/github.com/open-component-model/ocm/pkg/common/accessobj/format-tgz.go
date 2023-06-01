// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessobj

import (
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/compression"
)

var FormatTGZ = NewTarHandlerWithCompression(accessio.FormatTGZ, compression.Gzip)

func init() {
	RegisterFormat(FormatTGZ)
}
