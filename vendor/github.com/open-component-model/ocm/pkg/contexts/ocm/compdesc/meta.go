// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package compdesc

import (
	"time"

	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
)

type (
	ObjectMeta   = metav1.ObjectMeta
	Timestamp    = metav1.Timestamp
	Provider     = metav1.Provider
	ProviderName = metav1.ProviderName

	Labels      = metav1.Labels
	Label       = metav1.Label
	LabelOption = metav1.LabelOption

	Identity = metav1.Identity

	DigestSpec = metav1.DigestSpec

	ResourceRelation = metav1.ResourceRelation

	Signatures    = metav1.Signatures
	Signature     = metav1.Signature
	SignatureSpec = metav1.SignatureSpec
)

const (
	LocalRelation    = metav1.LocalRelation
	ExternalRelation = metav1.ExternalRelation

	NoDigest = metav1.NoDigest
)

func NewIdentity(name string, extras ...string) Identity {
	return metav1.NewIdentity(name, extras...)
}

func NewExtraIdentity(extras ...string) Identity {
	return metav1.NewExtraIdentity(extras...)
}

func IsIdentity(s string) bool {
	return metav1.IsIdentity(s)
}

func WithSigning(b ...bool) {
	metav1.WithSigning(b...)
}

func WithVersion(v string) {
	metav1.WithVersion(v)
}

func NewExcludeFromSignatureDigest() *DigestSpec {
	return metav1.NewExcludeFromSignatureDigest()
}

func NewTimestamp() Timestamp {
	return metav1.NewTimestamp()
}

func NewTimestampP() *Timestamp {
	return metav1.NewTimestampP()
}

func NewTimestampFor(t time.Time) Timestamp {
	return metav1.NewTimestampFor(t)
}

func NewTimestampPFor(t time.Time) *Timestamp {
	return metav1.NewTimestampPFor(t)
}
