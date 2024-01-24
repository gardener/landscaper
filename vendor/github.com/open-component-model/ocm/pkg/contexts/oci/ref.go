// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"fmt"
	"strings"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/contexts/oci/grammar"
	"github.com/open-component-model/ocm/pkg/errors"
)

// to find a suitable secret for images on Docker Hub, we need its two domains to do matching.
const (
	dockerHubDomain       = "docker.io"
	dockerHubLegacyDomain = "index.docker.io"

	KIND_OCI_REFERENCE       = "oci reference"
	KIND_ARETEFACT_REFERENCE = "artifact reference"
)

// ParseRepo parses a standard oci repository reference into a internal representation.
func ParseRepo(ref string) (UniformRepositorySpec, error) {
	create := false
	if strings.HasPrefix(ref, "+") {
		create = true
		ref = ref[1:]
	}
	match := grammar.AnchoredRegistryRegexp.FindSubmatch([]byte(ref))
	if match == nil {
		match = grammar.AnchoredGenericRegistryRegexp.FindSubmatch([]byte(ref))
		if match == nil {
			return UniformRepositorySpec{}, errors.ErrInvalid(KIND_OCI_REFERENCE, ref)
		}
		h := string(match[1])
		t, _ := grammar.SplitTypeSpec(h)
		return UniformRepositorySpec{
			Type:            t,
			TypeHint:        h,
			Info:            string(match[2]),
			CreateIfMissing: create,
		}, nil
	}
	h := string(match[1])
	t, _ := grammar.SplitTypeSpec(h)
	return UniformRepositorySpec{
		Type:            t,
		TypeHint:        h,
		Scheme:          string(match[2]),
		Host:            string(match[3]),
		CreateIfMissing: create,
	}, nil
}

// RefSpec is a go internal representation of an oci reference.
type RefSpec struct {
	UniformRepositorySpec `json:",inline"`
	ArtSpec               `json:",inline"`
}

func pointer(b []byte) *string {
	if len(b) == 0 {
		return nil
	}
	s := string(b)
	return &s
}

func dig(b []byte) *digest.Digest {
	if len(b) == 0 {
		return nil
	}
	s := digest.Digest(b)
	return &s
}

// ParseRef parses a oci reference into a internal representation.
func ParseRef(ref string) (RefSpec, error) {
	create := false
	if strings.HasPrefix(ref, "+") {
		create = true
		ref = ref[1:]
	}

	spec := RefSpec{UniformRepositorySpec: UniformRepositorySpec{CreateIfMissing: create}}

	match := grammar.FileReferenceRegexp.FindSubmatch([]byte(ref))
	if match != nil {
		spec.Type = string(match[1])
		spec.Info = string(match[2])
		spec.Repository = string(match[3])
		spec.Tag = pointer(match[4])
		spec.Digest = dig(match[5])
		return spec, nil
	}
	match = grammar.DockerLibraryReferenceRegexp.FindSubmatch([]byte(ref))
	if match != nil {
		spec.Host = dockerHubDomain
		spec.Repository = "library" + grammar.RepositorySeparator + string(match[1])
		spec.Tag = pointer(match[2])
		spec.Digest = dig(match[3])
		return spec, nil
	}
	match = grammar.DockerReferenceRegexp.FindSubmatch([]byte(ref))
	if match != nil {
		spec.Host = dockerHubDomain
		spec.Repository = string(match[1])
		spec.Tag = pointer(match[2])
		spec.Digest = dig(match[3])
		return spec, nil
	}
	match = grammar.ReferenceRegexp.FindSubmatch([]byte(ref))
	if match != nil {
		spec.Scheme = string(match[1])
		spec.Host = string(match[2])
		spec.Repository = string(match[3])
		spec.Tag = pointer(match[4])
		spec.Digest = dig(match[5])
		return spec, nil
	}
	match = grammar.TypedReferenceRegexp.FindSubmatch([]byte(ref))
	if match != nil {
		spec.Type = string(match[1])
		spec.Scheme = string(match[2])
		spec.Host = string(match[3])
		spec.Repository = string(match[4])
		spec.Tag = pointer(match[5])
		spec.Digest = dig(match[6])
		return spec, nil
	}
	match = grammar.TypedURIRegexp.FindSubmatch([]byte(ref))
	if match != nil {
		spec.Type = string(match[1])
		spec.Scheme = string(match[2])
		spec.Host = string(match[3])
		spec.Repository = string(match[4])
		spec.Tag = pointer(match[5])
		spec.Digest = dig(match[6])
		return spec, nil
	}
	match = grammar.TypedGenericReferenceRegexp.FindSubmatch([]byte(ref))
	if match != nil {
		spec.Type = string(match[1])
		spec.Info = string(match[2])
		spec.Repository = string(match[3])
		spec.Tag = pointer(match[4])
		spec.Digest = dig(match[5])
		return spec, nil
	}
	match = grammar.AnchoredRegistryRegexp.FindSubmatch([]byte(ref))
	if match != nil {
		spec.Type = string(match[1])
		spec.Info = string(match[2])
		spec.Repository = string(match[3])
		spec.Tag = pointer(match[4])
		spec.Digest = dig(match[5])
		return spec, nil
	}

	match = grammar.AnchoredGenericRegistryRegexp.FindSubmatch([]byte(ref))
	if match != nil {
		spec.Type = string(match[1])
		spec.Info = string(match[2])

		match = grammar.ErrorCheckRegexp.FindSubmatch([]byte(ref))
		if match != nil {
			if len(match[3]) != 0 || len(match[4]) != 0 {
				return RefSpec{}, errors.ErrInvalid(KIND_OCI_REFERENCE, ref)
			}
		}
		return spec, nil
	}
	return RefSpec{}, errors.ErrInvalid(KIND_OCI_REFERENCE, ref)
}

func (r *RefSpec) Name() string {
	return r.UniformRepositorySpec.ComposeRef(r.Repository)
}

func (r *RefSpec) String() string {
	art := r.Repository
	if r.Tag != nil {
		art = fmt.Sprintf("%s:%s", art, *r.Tag)
	}
	if r.Digest != nil {
		art = fmt.Sprintf("%s@%s", art, r.Digest.String())
	}
	return r.UniformRepositorySpec.ComposeRef(art)
}

// CredHost fallback to legacy docker domain if applicable
// this is how containerd translates the old domain for DockerHub to the new one, taken from containerd/reference/docker/reference.go:674.
func (r *RefSpec) CredHost() string {
	if r.Host == dockerHubDomain {
		return dockerHubLegacyDomain
	}
	return r.Host
}

func (r RefSpec) DeepCopy() RefSpec {
	if r.Tag != nil {
		tag := *r.Tag
		r.Tag = &tag
	}
	if r.Digest != nil {
		dig := *r.Digest
		r.Digest = &dig
	}
	return r
}

////////////////////////////////////////////////////////////////////////////////

func ParseArt(art string) (ArtSpec, error) {
	match := grammar.AnchoredArtifactVersionRegexp.FindSubmatch([]byte(art))

	if match == nil {
		return ArtSpec{}, errors.ErrInvalid(KIND_ARETEFACT_REFERENCE, art)
	}
	var tag *string
	var dig *digest.Digest

	if match[2] != nil {
		t := string(match[2])
		tag = &t
	}
	if match[3] != nil {
		t := string(match[3])
		d, err := digest.Parse(t)
		if err != nil {
			return ArtSpec{}, errors.ErrInvalidWrap(err, KIND_ARETEFACT_REFERENCE, art)
		}
		dig = &d
	}
	return ArtSpec{
		Repository: string(match[1]),
		Tag:        tag,
		Digest:     dig,
	}, nil
}

// ArtSpec is a go internal representation of a oci reference.
type ArtSpec struct {
	// Repository is the part of a reference without its hostname
	Repository string `json:"repository"`
	// +optional
	Tag *string `json:"tag,omitempty"`
	// +optional
	Digest *digest.Digest `json:"digest,omitempty"`
}

func (r *ArtSpec) Version() string {
	if r.Tag != nil {
		return *r.Tag
	}
	if r.Digest != nil {
		return "@" + string(*r.Digest)
	}
	return "latest"
}

func (r *ArtSpec) IsRegistry() bool {
	return r.Repository == ""
}

func (r *ArtSpec) IsVersion() bool {
	return r.Tag != nil || r.Digest != nil
}

func (r *ArtSpec) IsTagged() bool {
	return r.Tag != nil
}

func (r *ArtSpec) Reference() string {
	if r.Tag != nil {
		return *r.Tag
	}
	if r.Digest != nil {
		return "@" + string(*r.Digest)
	}
	return "latest"
}

func (r *ArtSpec) String() string {
	s := r.Repository
	if r.Tag != nil {
		s += fmt.Sprintf(":%s", *r.Tag)
	}
	if r.Digest != nil {
		s += fmt.Sprintf("@%s", r.Digest.String())
	}
	return s
}
