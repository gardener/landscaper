// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package downloader

import "io"

// Downloader defines a downloader for various objects using a WriterAt to
// transfer data to.
type Downloader interface {
	Download(w io.WriterAt) error
}
