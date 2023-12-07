// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blobaccess

import (
	"io"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/utils"
)

func ForCachedBlobAccess(blob BlobAccess, fss ...vfs.FileSystem) (BlobAccess, error) {
	fs := utils.FileSystem(fss...)

	r, err := blob.Reader()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	file, err := vfs.TempFile(fs, "", "cachedBlob*")
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(file, r)
	if err != nil {
		return nil, err
	}
	file.Close()

	return ForTemporaryFilePath(blob.MimeType(), file.Name(), fs), nil
}
