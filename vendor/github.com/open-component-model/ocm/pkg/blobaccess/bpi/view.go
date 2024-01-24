// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bpi

import (
	"fmt"
	"io"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/refmgmt"
	"github.com/open-component-model/ocm/pkg/utils"
)

func NewBlobAccessForBase(acc BlobAccessBase, closer ...io.Closer) BlobAccess {
	return refmgmt.WithView[BlobAccessBase, BlobAccess](acc, blobAccessViewCreator, closer...)
}

func blobAccessViewCreator(blob BlobAccessBase, view *refmgmt.View[BlobAccess]) BlobAccess {
	return &blobAccessView{view, blob}
}

type blobAccessView struct {
	*refmgmt.View[BlobAccess]
	baseblob BlobAccessBase
}

var (
	_ utils.Validatable = (*blobAccessView)(nil)
	_ utils.Unwrappable = (*blobAccessView)(nil)
)

func (b *blobAccessView) base() BlobAccessBase {
	return b.baseblob
}

func (b *blobAccessView) Unwrap() interface{} {
	return b.baseblob
}

func (b *blobAccessView) Close() error {
	return b.View.Close()
}

func (b *blobAccessView) Validate() error {
	return utils.ValidateObject(b.baseblob)
}

func (b *blobAccessView) Get() (result []byte, err error) {
	return result, b.Execute(func() error {
		result, err = b.baseblob.Get()
		if err != nil {
			return err
		}
		return nil
	})
}

func (b *blobAccessView) Reader() (result io.ReadCloser, err error) {
	return result, b.Execute(func() error {
		result, err = b.baseblob.Reader()
		if err != nil {
			return fmt.Errorf("unable to read access: %w", err)
		}

		return nil
	})
}

func (b *blobAccessView) Digest() (result digest.Digest) {
	err := b.Execute(func() error {
		result = b.baseblob.Digest()
		return nil
	})
	if err != nil {
		return BLOB_UNKNOWN_DIGEST
	}
	return
}

func (b *blobAccessView) MimeType() string {
	return b.baseblob.MimeType()
}

func (b *blobAccessView) DigestKnown() bool {
	return b.baseblob.DigestKnown()
}

func (b *blobAccessView) Size() (result int64) {
	err := b.Execute(func() error {
		result = b.baseblob.Size()
		return nil
	})
	if err != nil {
		return BLOB_UNKNOWN_SIZE
	}
	return
}
