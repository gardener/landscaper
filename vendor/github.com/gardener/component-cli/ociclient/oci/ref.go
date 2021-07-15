// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"fmt"
	"strings"

	dockerreference "github.com/containerd/containerd/reference/docker"
	"github.com/opencontainers/go-digest"
)

// to find a suitable secret for images on Docker Hub, we need its two domains to do matching
const (
	dockerHubDomain       = "docker.io"
	dockerHubLegacyDomain = "index.docker.io"
)

// ParseRef parses a oci reference into a internal representation.
func ParseRef(ref string) (RefSpec, error) {
	if strings.Contains(ref, "://") {
		// remove protocol if exists
		i := strings.Index(ref, "://") + 3
		ref = ref[i:]
	}

	parsedRef, err := dockerreference.ParseDockerRef(ref)
	if err != nil {
		return RefSpec{}, err
	}

	spec := RefSpec{
		Host:       dockerreference.Domain(parsedRef),
		Repository: dockerreference.Path(parsedRef),
	}

	switch r := parsedRef.(type) {
	case dockerreference.Tagged:
		tag := r.Tag()
		spec.Tag = &tag
	case dockerreference.Digested:
		d := r.Digest()
		spec.Digest = &d
	}

	// fallback to legacy docker domain if applicable
	// this is how containerd translates the old domain for DockerHub to the new one, taken from containerd/reference/docker/reference.go:674
	if spec.Host == dockerHubDomain {
		spec.Host = dockerHubLegacyDomain
	}
	return spec, nil
}

// RefSpec is a go internal representation of a oci reference.
type RefSpec struct {
	// Host is the hostname of a oci ref.
	Host string
	// Repository is the part of a reference without its hostname
	Repository string
	// +optional
	Tag *string
	// +optional
	Digest *digest.Digest
}

func (r *RefSpec) Name() string {
	return fmt.Sprintf("%s/%s", r.Host, r.Repository)
}

func (r RefSpec) String() string {
	if r.Tag != nil {
		return fmt.Sprintf("%s:%s", r.Name(), *r.Tag)
	}
	if r.Digest != nil {
		return fmt.Sprintf("%s@%s", r.Name(), r.Digest.String())
	}
	return ""
}

func (r RefSpec) DeepCopy() RefSpec {
	refspec := RefSpec{
		Host:       r.Host,
		Repository: r.Repository,
	}
	if r.Tag != nil {
		tag := *r.Tag
		refspec.Tag = &tag
	}
	if r.Digest != nil {
		dig := r.Digest.String()
		d := digest.FromString(dig)
		refspec.Digest = &d
	}
	return refspec
}
