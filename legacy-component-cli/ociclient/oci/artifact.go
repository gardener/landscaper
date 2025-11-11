// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"encoding/json"
	"errors"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Artifact represents an OCI artifact/repository.
// It can either be a single manifest or an index (manifest list).
type Artifact struct {
	manifest *Manifest
	index    *Index
}

// Manifest represents am OCI image manifest
type Manifest struct {
	Descriptor ocispecv1.Descriptor
	Data       *ocispecv1.Manifest
}

// Index represents an OCI image index
type Index struct {
	Manifests   []*Manifest
	Annotations map[string]string
}

// NewManifestArtifact creates a new OCI artifact of type manifest
func NewManifestArtifact(m *Manifest) (*Artifact, error) {
	a := Artifact{}
	return &a, a.SetManifest(m)
}

// NewIndexArtifact creates a new OCI artifact of type index
func NewIndexArtifact(i *Index) (*Artifact, error) {
	a := Artifact{}
	return &a, a.SetIndex(i)
}

// GetManifest returns the manifest property
func (a *Artifact) GetManifest() *Manifest {
	return a.manifest
}

// GetManifest returns the index property
func (a *Artifact) GetIndex() *Index {
	return a.index
}

// SetManifest sets the manifest property.
// If the OCI artifact is of type index, an error is returned.
func (a *Artifact) SetManifest(m *Manifest) error {
	if m == nil {
		return errors.New("manifest must not be nil")
	}

	if a.IsIndex() {
		return errors.New("unable to set manifest on index artifact")
	}

	a.manifest = m
	return nil
}

// SetIndex sets the index property.
// If the OCI artifact is of type manifest, an error is returned.
func (a *Artifact) SetIndex(i *Index) error {
	if i == nil {
		return errors.New("index must not be nil")
	}

	if a.IsManifest() {
		return errors.New("unable to set index on manifest artifact")
	}

	a.index = i
	return nil
}

func (a *Artifact) IsManifest() bool {
	return a.manifest != nil
}

func (a *Artifact) IsIndex() bool {
	return a.index != nil
}

func (a *Artifact) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Manifest *Manifest `json:"manifest"`
		Index    *Index    `json:"index"`
	}{
		Manifest: a.manifest,
		Index:    a.index,
	})
}
