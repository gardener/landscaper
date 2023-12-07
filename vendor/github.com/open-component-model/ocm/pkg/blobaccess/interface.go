// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blobaccess

import (
	"github.com/open-component-model/ocm/pkg/blobaccess/internal"
)

const (
	KIND_BLOB      = internal.KIND_BLOB
	KIND_MEDIATYPE = internal.KIND_MEDIATYPE

	BLOB_UNKNOWN_SIZE   = internal.BLOB_UNKNOWN_SIZE
	BLOB_UNKNOWN_DIGEST = internal.BLOB_UNKNOWN_DIGEST
)

type (
	DataAccess = internal.DataAccess
	DataReader = internal.DataReader
	DataGetter = internal.DataGetter
)

type (
	BlobAccess         = internal.BlobAccess
	BlobAccessProvider = internal.BlobAccessProvider

	DigestSource = internal.DigestSource
	MimeType     = internal.MimeType
)

type FileLocation = internal.FileLocation
