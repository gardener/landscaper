// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"github.com/open-component-model/ocm/pkg/utils"
)

type ModificationOption interface {
	ApplyModificationOption(opts *ModificationOptions)
}

type ModificationOptions struct {
	// ModifyResource disables the modification of signature releveant
	// resource parts.
	ModifyResource *bool

	// AcceptExistentDigests don't validate/recalculate the content digest
	// of resources.
	AcceptExistentDigests *bool

	// DefaultHashAlgorithm is the hash algorithm to use if no specific setting os found
	DefaultHashAlgorithm string

	// HasherProvider is the factory for hash algorithms to use.
	HasherProvider HasherProvider

	// SkipVerify disabled the verification of given digests
	SkipVerify *bool

	// SkipDigest disabled digest creation (for legacy code, only!)
	SkipDigest *bool
}

func (m *ModificationOptions) IsModifyResource() bool {
	return utils.AsBool(m.ModifyResource)
}

func (m *ModificationOptions) IsAcceptExistentDigests() bool {
	return utils.AsBool(m.AcceptExistentDigests)
}

func (m *ModificationOptions) IsSkipDigest() bool {
	return utils.AsBool(m.SkipDigest)
}

func (m *ModificationOptions) IsSkipVerify() bool {
	return utils.AsBool(m.SkipVerify)
}

func (m *ModificationOptions) ApplyModificationOptions(list ...ModificationOption) *ModificationOptions {
	for _, o := range list {
		if o != nil {
			o.ApplyModificationOption(m)
		}
	}
	return m
}

func (m ModificationOptions) ApplyModificationOption(opts *ModificationOptions) {
	applyBool(m.ModifyResource, &opts.ModifyResource)
	applyBool(m.AcceptExistentDigests, &opts.AcceptExistentDigests)
	applyBool(m.SkipDigest, &opts.SkipDigest)
	applyBool(m.SkipVerify, &opts.SkipVerify)
	if m.HasherProvider != nil {
		opts.HasherProvider = m.HasherProvider
	}
	if m.DefaultHashAlgorithm != "" {
		opts.DefaultHashAlgorithm = m.DefaultHashAlgorithm
	}
}

func applyBool(m *bool, t **bool) {
	if m != nil {
		*t = utils.BoolP(*m)
	}
}

func (m *ModificationOptions) GetHasher(algo ...string) Hasher {
	return m.HasherProvider.GetHasher(utils.OptionalDefaulted(m.DefaultHashAlgorithm, algo...))
}

func NewModificationOptions(list ...ModificationOption) *ModificationOptions {
	var m ModificationOptions
	m.ApplyModificationOptions(list...)
	return &m
}

////////////////////////////////////////////////////////////////////////////////

type modifyresource bool

func (m modifyresource) ApplyModificationOption(opts *ModificationOptions) {
	opts.ModifyResource = utils.BoolP(m)
}

func ModifyResource(flag ...bool) ModificationOption {
	return modifyresource(utils.OptionalDefaultedBool(true, flag...))
}

////////////////////////////////////////////////////////////////////////////////

type acceptdigests bool

func (m acceptdigests) ApplyModificationOption(opts *ModificationOptions) {
	opts.AcceptExistentDigests = utils.BoolP(m)
}

func AcceptExistentDigests(flag ...bool) ModificationOption {
	return acceptdigests(utils.OptionalDefaultedBool(true, flag...))
}

////////////////////////////////////////////////////////////////////////////////

type hashalgo string

func (m hashalgo) ApplyModificationOption(opts *ModificationOptions) {
	opts.DefaultHashAlgorithm = string(m)
}

func WithDefaultHashAlgorithm(algo ...string) ModificationOption {
	return hashalgo(utils.Optional(algo...))
}

////////////////////////////////////////////////////////////////////////////////

type hashprovider struct {
	prov HasherProvider
}

func (m *hashprovider) ApplyModificationOption(opts *ModificationOptions) {
	opts.HasherProvider = m.prov
}

func WithHasherProvider(prov HasherProvider) ModificationOption {
	return &hashprovider{prov}
}

////////////////////////////////////////////////////////////////////////////////

type skipverify bool

func (m skipverify) ApplyModificationOption(opts *ModificationOptions) {
	opts.SkipVerify = utils.BoolP(m)
}

func SkipVerify(flag ...bool) ModificationOption {
	return skipverify(utils.OptionalDefaultedBool(true, flag...))
}

////////////////////////////////////////////////////////////////////////////////

type skipdigest bool

func (m skipdigest) ApplyModificationOption(opts *ModificationOptions) {
	opts.SkipDigest = utils.BoolP(m)
}

// SkipDigest disables digest creation if enabled.
//
// Deprecated: for legacy code, only.
func SkipDigest(flag ...bool) ModificationOption {
	return skipdigest(utils.OptionalDefaultedBool(true, flag...))
}
