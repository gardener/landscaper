// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/equivalent"
	"github.com/open-component-model/ocm/pkg/signing"
)

const (
	// ExcludeFromSignature used in digest field for normalisationAlgorithm (in combination with NoDigest for hashAlgorithm and value)
	// to indicate the resource content should not be part of the signature.
	ExcludeFromSignature = "EXCLUDE-FROM-SIGNATURE"

	// NoDigest used in digest field for hashAlgorithm and value (in combination with ExcludeFromSignature for normalisationAlgorithm)
	// to indicate the resource content should not be part of the signature.
	NoDigest = "NO-DIGEST"
)

// Signatures is a list of signatures.
type Signatures []Signature

func (s Signatures) Equivalent(o Signatures) equivalent.EqualState {
	if len(s) != len(o) {
		return equivalent.StateNotEquivalent()
	}
outer:
	for _, a := range s {
		if b := o.GetByName(a.Name); b != nil {
			if reflect.DeepEqual(&a, b) {
				continue outer
			}
		}
		return equivalent.StateNotEquivalent()
	}
	return equivalent.StateEquivalent()
}

func (s Signatures) Len() int {
	return len(s)
}

func (s Signatures) Get(i int) *Signature {
	if i >= 0 && i < len(s) {
		return &s[i]
	}
	return nil
}

func (s Signatures) Copy() Signatures {
	if s == nil {
		return nil
	}
	out := make(Signatures, s.Len())
	for i, v := range s {
		out[i] = *v.Copy()
	}
	return out
}

func (s Signatures) GetIndex(name string) int {
	for i, v := range s {
		if v.Name == name {
			return i
		}
	}
	return -1
}

func (s Signatures) GetByName(name string) *Signature {
	if idx := s.GetIndex(name); idx >= 0 {
		return &s[idx]
	}
	return nil
}

func (s *Signatures) Set(sig Signature) {
	if idx := s.GetIndex(sig.Name); idx < 0 {
		*s = append(*s, sig)
	} else {
		(*s)[idx] = sig
	}
}

// DigestSpec defines a digest.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type DigestSpec struct {
	HashAlgorithm          string `json:"hashAlgorithm"`
	NormalisationAlgorithm string `json:"normalisationAlgorithm"`
	Value                  string `json:"value"`
}

func (d *DigestSpec) String() string {
	return fmt.Sprintf("%s:%s[%s]", d.HashAlgorithm, d.Value, d.NormalisationAlgorithm)
}

func (d *DigestSpec) IsComplete() bool {
	return d != nil && d.HashAlgorithm != "" && d.NormalisationAlgorithm != "" && d.Value != ""
}

func (d *DigestSpec) IsNone() bool {
	return d == nil || *d == DigestSpec{}
}

func (d *DigestSpec) IsExcluded() bool {
	return d != nil && *d == excluded
}

// Copy provides a copy of the digest spec.
func (d *DigestSpec) Copy() *DigestSpec {
	if d == nil {
		return nil
	}
	r := *d
	return &r
}

func (d *DigestSpec) Equal(o *DigestSpec) bool {
	if d == o {
		return true
	}
	if d == nil || o == nil {
		return false
	}
	return *d == *o
}

func (d *DigestSpec) Equivalent(o *DigestSpec) equivalent.EqualState {
	if d == nil {
		d, o = o, d
	}
	if d == nil {
		return equivalent.StateNotArtifactEqual(false)
	}

	if (d.IsExcluded() && o == nil) || reflect.DeepEqual(d, o) {
		return equivalent.StateEquivalent()
	}
	return equivalent.StateNotArtifactEqual(d != nil && o != nil)
}

// SignatureSpec defines a signature.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type SignatureSpec struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
	MediaType string `json:"mediaType"`
	Issuer    string `json:"issuer,omitempty"`
}

// ConvertToSigning converts a cd signature to a signing signature.
func (s *SignatureSpec) ConvertToSigning() *signing.Signature {
	return &signing.Signature{
		Value:     s.Value,
		MediaType: s.MediaType,
		Algorithm: s.Algorithm,
		Issuer:    s.Issuer,
	}
}

// Signature defines a digest and corresponding signature, identifiable by name.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type Signature struct {
	Name      string        `json:"name"`
	Digest    DigestSpec    `json:"digest"`
	Signature SignatureSpec `json:"signature"`
}

// Copy provides a copy of the signature data.
func (s *Signature) Copy() *Signature {
	if s == nil {
		return nil
	}
	r := *s
	return &r
}

// ConvertToSigning converts a cd signature to a signing signature.
func (s *Signature) ConvertToSigning() *signing.Signature {
	return s.Signature.ConvertToSigning()
}

func SignatureSpecFor(s *signing.Signature) *SignatureSpec {
	return &SignatureSpec{
		Value:     s.Value,
		MediaType: s.MediaType,
		Algorithm: s.Algorithm,
		Issuer:    s.Issuer,
	}
}

var excluded = DigestSpec{
	HashAlgorithm:          NoDigest,
	NormalisationAlgorithm: ExcludeFromSignature,
	Value:                  NoDigest,
}

// NewExcludeFromSignatureDigest returns the special digest notation to indicate the resource content should not be part of the signature.
func NewExcludeFromSignatureDigest() *DigestSpec {
	e := excluded
	return &e
}

////////////////////////////////////////////////////////////////////////////////

// NestedDigests defines a list of nested components.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type NestedDigests []NestedComponentDigests

func (d NestedDigests) String() string {
	r := ""
	sep := ""
	for _, e := range d {
		r = fmt.Sprintf("%s%s%s", r, sep, e.String())
		sep = "\n"
	}
	return r
}

func (d NestedDigests) Copy() NestedDigests {
	if d == nil {
		return nil
	}
	r := make([]NestedComponentDigests, len(d))
	for i, e := range d {
		r[i] = *e.Copy()
	}
	return r
}

func (d NestedDigests) Lookup(name, version string) *NestedComponentDigests {
	for _, e := range d {
		if e.Name == name && e.Version == version {
			return &e
		}
	}
	return nil
}

// NestedComponentDigests defines nested components.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type NestedComponentDigests struct {
	Name      string          `json:"name"`
	Version   string          `json:"version"`
	Digest    *DigestSpec     `json:"digest,omitempty"`
	Resources ArtefactDigests `json:"resourceDigests,omitempty"`
}

func (d *NestedComponentDigests) String() string {
	r := []string{
		fmt.Sprintf("%s:%s: %s", d.Name, d.Version, d.Digest.String()),
	}
	for _, a := range d.Resources {
		r = append(r, "  "+a.String())
	}
	return strings.Join(r, "\n")
}

func (d *NestedComponentDigests) Lookup(name, version string, extra Identity) *ArtefactDigest {
	if d == nil {
		return nil
	}
	return d.Resources.Lookup(name, version, extra)
}

func (d *NestedComponentDigests) Copy() *NestedComponentDigests {
	if d == nil {
		return nil
	}
	r := *d
	r.Digest = d.Digest.Copy()
	r.Resources = make([]ArtefactDigest, len(d.Resources))
	for i, e := range d.Resources {
		r.Resources[i] = *e.Copy()
	}
	return &r
}

// ArtefactDigests defines a list of artefact digest information.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type ArtefactDigests []ArtefactDigest

func (d ArtefactDigests) Lookup(name, version string, extra Identity) *ArtefactDigest {
	for _, e := range d {
		if e.Name == name && e.Version == version && e.ExtraIdentity.Equals(extra) {
			return &e
		}
	}
	return nil
}

func (d ArtefactDigests) Match(o ArtefactDigests) bool {
	if len(d) != len(o) {
		return false
	}
	for _, e := range d {
		i := o.Lookup(e.Name, e.Version, e.ExtraIdentity)
		if i == nil || i.Digest != e.Digest {
			return false
		}
	}
	return true
}

// ArtefactDigest defines artefact digest information.
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type ArtefactDigest struct {
	Name          string     `json:"name"`
	Version       string     `json:"version"`
	ExtraIdentity Identity   `json:"extraIdentity,omitempty"`
	Digest        DigestSpec `json:"digest"`
}

func (d *ArtefactDigest) Copy() *ArtefactDigest {
	r := *d
	r.ExtraIdentity = d.ExtraIdentity.Copy()
	return &r
}

func (d *ArtefactDigest) String() string {
	return fmt.Sprintf("%s:%s[%s]: %s", d.Name, d.Version, d.ExtraIdentity.String(), d.Digest.String())
}
