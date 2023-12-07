// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bpi

import (
	"github.com/open-component-model/ocm/pkg/blobaccess/internal"
	"github.com/open-component-model/ocm/pkg/refmgmt"
)

const (
	KIND_BLOB      = internal.KIND_BLOB
	KIND_MEDIATYPE = internal.KIND_MEDIATYPE

	BLOB_UNKNOWN_SIZE   = internal.BLOB_UNKNOWN_SIZE
	BLOB_UNKNOWN_DIGEST = internal.BLOB_UNKNOWN_DIGEST
)

var ErrClosed = refmgmt.ErrClosed

type DataAccess = internal.DataAccess

type (
	BlobAccess         = internal.BlobAccess
	BlobAccessBase     = internal.BlobAccessBase
	BlobAccessProvider = internal.BlobAccessProvider

	DigestSource = internal.DigestSource
	MimeType     = internal.MimeType
)

type FileLocation = internal.FileLocation

type BlobAccessProviderFunction func() (BlobAccess, error)

func (p BlobAccessProviderFunction) BlobAccess() (BlobAccess, error) {
	return p()
}
