// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/hashattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/signingattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/internal"
	"github.com/open-component-model/ocm/pkg/signing/hasher/sha256"
)

type (
	ModificationOption  = internal.ModificationOption
	ModificationOptions = internal.ModificationOptions
)

func NewModificationOptions(list ...ModificationOption) *ModificationOptions {
	return internal.NewModificationOptions(list...)
}

func ModifyResource(flag ...bool) ModificationOption {
	return internal.ModifyResource(flag...)
}

func AcceptExistentDigests(flag ...bool) ModificationOption {
	return internal.AcceptExistentDigests(flag...)
}

func WithDefaultHashAlgorithm(algo ...string) ModificationOption {
	return internal.WithDefaultHashAlgorithm(algo...)
}

func WithHasherProvider(prov HasherProvider) ModificationOption {
	return internal.WithHasherProvider(prov)
}

func SkipVerify(flag ...bool) ModificationOption {
	return internal.SkipVerify(flag...)
}

// SkipDigest disables digest creation if enabled.
//
// Deprecated: for legacy code, only.
func SkipDigest(flag ...bool) ModificationOption {
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
