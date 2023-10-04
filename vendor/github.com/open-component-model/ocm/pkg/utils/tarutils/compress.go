// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package tarutils

import (
	"compress/gzip"
	"io"
)

func Gzip(w io.Writer) io.WriteCloser {
	return gzip.NewWriter(w)
}
