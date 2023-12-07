// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/hashattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/signingattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/internal"
	"github.com/open-component-model/ocm/pkg/signing/hasher/sha256"
)

type (
	ModificationOption  = internal.ModificationOption
	ModificationOptions = internal.ModificationOptions

	BlobModificationOption  = internal.BlobModificationOption
	BlobModificationOptions = internal.BlobModificationOptions

	BlobUploadOption  = internal.BlobUploadOption
	BlobUploadOptions = internal.BlobUploadOptions

	AddVersionOption  = internal.AddVersionOption
	AddVersionOptions = internal.AddVersionOptions
)

////////////////////////////////////////////////////////////////////////////////

func NewAddVersionOptions(list ...AddVersionOption) *AddVersionOptions {
	return internal.NewAddVersionOptions(list...)
}

// Overwrite enabled the overwrite mode for adding a component version.
func Overwrite(flag ...bool) AddVersionOption {
	return internal.Overwrite(flag...)
}

////////////////////////////////////////////////////////////////////////////////

func NewBlobModificationOptions(list ...BlobModificationOption) *BlobModificationOptions {
	return internal.NewBlobModificationOptions(list...)
}

////////////////////////////////////////////////////////////////////////////////

func NewBlobUploadOptions(list ...BlobUploadOption) *BlobUploadOptions {
	return internal.NewBlobUploadOptions(list...)
}

func UseBlobHandlers(h BlobHandlerProvider) internal.BlobOptionImpl {
	return internal.UseBlobHandlers(h)
}

////////////////////////////////////////////////////////////////////////////////

func NewModificationOptions(list ...ModificationOption) *ModificationOptions {
	return internal.NewModificationOptions(list...)
}

func ModifyResource(flag ...bool) internal.ModOptionImpl {
	return internal.ModifyResource(flag...)
}

func AcceptExistentDigests(flag ...bool) internal.ModOptionImpl {
	return internal.AcceptExistentDigests(flag...)
}

func WithDefaultHashAlgorithm(algo ...string) internal.ModOptionImpl {
	return internal.WithDefaultHashAlgorithm(algo...)
}

func WithHasherProvider(prov HasherProvider) internal.ModOptionImpl {
	return internal.WithHasherProvider(prov)
}

func SkipVerify(flag ...bool) internal.ModOptionImpl {
	return internal.SkipVerify(flag...)
}

// SkipDigest disables digest creation if enabled.
//
// Deprecated: for legacy code, only.
func SkipDigest(flag ...bool) internal.ModOptionImpl {
	return internal.SkipDigest(flag...)
}

///////////////////////////////////////////////////////

func CompleteModificationOptions(ctx ContextProvider, m *ModificationOptions) {
	attr := hashattr.Get(ctx.OCMContext())
	if m.DefaultHashAlgorithm == "" {
		m.DefaultHashAlgorithm = attr.DefaultHasher
	}
	if m.DefaultHashAlgorithm == "" {
		m.DefaultHashAlgorithm = sha256.Algorithm
	}
	if m.HasherProvider == nil {
		m.HasherProvider = signingattr.Get(ctx.OCMContext())
	}
}
