// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessio

import (
	"io"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/iotools"
)

// Deprecated: use iotools.DigestWriter.
type DigestWriter = iotools.DigestWriter

// Deprecated: use iotools.NewDefaultDigestWriter.
func NewDefaultDigestWriter(w io.WriteCloser) *iotools.DigestWriter {
	return iotools.NewDefaultDigestWriter(w)
}

// Deprecated: use iotools.NewDigestWriterWith.
func NewDigestWriterWith(algorithm digest.Algorithm, w io.WriteCloser) *iotools.DigestWriter {
	return iotools.NewDigestWriterWith(algorithm, w)
}
