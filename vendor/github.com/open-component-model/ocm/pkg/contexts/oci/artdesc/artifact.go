// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package artdesc

import (
	"encoding/json"

	"github.com/containerd/containerd/images"
	"github.com/opencontainers/go-digest"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc/helper"
	"github.com/open-component-model/ocm/pkg/errors"
)

const SchemeVersion = helper.SchemeVersion

const (
	MediaTypeImageManifest  = ociv1.MediaTypeImageManifest
	MediaTypeImageIndex     = ociv1.MediaTypeImageIndex
	MediaTypeImageLayer     = ociv1.MediaTypeImageLayer
	MediaTypeImageLayerGzip = ociv1.MediaTypeImageLayerGzip

	MediaTypeDockerSchema2Manifest     = images.MediaTypeDockerSchema2Manifest
	MediaTypeDockerSchema2ManifestList = images.MediaTypeDockerSchema2ManifestList

	MediaTypeImageConfig = ociv1.MediaTypeImageConfig
)

var legacy = false

type (
	Descriptor = ociv1.Descriptor
	Platform   = ociv1.Platform
)

type BlobDescriptorSource interface {
	GetBlobDescriptor(digest.Digest) *Descriptor
	MimeType() string
	IsValid() bool
}

// Artifact is the unified representation of an OCI artifact
// according to https://github.com/opencontainers/image-spec/blob/main/manifest.md
// It is either an image manifest or an image index manifest (fat image).
type Artifact struct {
	manifest *Manifest
	index    *Index
}

var (
	_ BlobDescriptorSource = (*Artifact)(nil)
	_ json.Marshaler       = (*Artifact)(nil)
	_ json.Unmarshaler     = (*Artifact)(nil)
)

func New() *Artifact {
	return &Artifact{}
}

func NewManifestArtifact() *Artifact {
	a := New()
	a.SetManifest(NewManifest())
	return a
}

func NewIndexArtifact() *Artifact {
	a := New()
	a.SetIndex(NewIndex())
	return a
}

func (d *Artifact) MimeType() string {
	if d.IsIndex() {
		return d.index.MimeType()
	}
	if d.IsManifest() {
		return d.manifest.MimeType()
	}
	return ""
}

func (d *Artifact) SetManifest(m *Manifest) error {
	if d.IsIndex() || d.IsManifest() {
		return errors.Newf("artifact descriptor already instantiated")
	}
	d.manifest = m
	return nil
}

func (d *Artifact) SetIndex(i *Index) error {
	if d.IsIndex() || d.IsManifest() {
		return errors.Newf("artifact descriptor already instantiated")
	}
	d.index = i
	return nil
}

func (d *Artifact) IsValid() bool {
	return d.manifest != nil || d.index != nil
}

func (d *Artifact) IsManifest() bool {
	return d.manifest != nil
}

func (d *Artifact) IsIndex() bool {
	return d.index != nil
}

func (d *Artifact) Index() *Index {
	return d.index
}

func (d *Artifact) Manifest() *Manifest {
	return d.manifest
}

func (d *Artifact) SetAnnotation(name, value string) error {
	return d.modifyAnnotation(func(annos *map[string]string) {
		if *annos == nil {
			*annos = map[string]string{}
		}
		(*annos)[name] = value
	})
}

func (d *Artifact) DeleteAnnotation(name string) error {
	return d.modifyAnnotation(func(annos *map[string]string) {
		if *annos == nil {
			return
		}
		delete(*annos, name)
		if len(*annos) == 0 {
			*annos = nil
		}
	})
}

func (d *Artifact) modifyAnnotation(mod func(annos *map[string]string)) error {
	var annos map[string]string

	switch {
	case d.manifest != nil:
		annos = d.manifest.Annotations
	case d.index != nil:
		annos = d.index.Annotations
	default:
		return errors.Newf("void artifact access")
	}
	mod(&annos)
	if d.manifest != nil {
		d.manifest.Annotations = annos
	} else {
		d.index.Annotations = annos
	}
	return nil
}

func (d *Artifact) ToBlobAccess() (accessio.BlobAccess, error) {
	if d.IsManifest() {
		return d.manifest.ToBlobAccess()
	}
	if d.IsIndex() {
		return d.index.ToBlobAccess()
	}
	return nil, errors.ErrInvalid("artifact descriptor")
}

func (d *Artifact) GetBlobDescriptor(digest digest.Digest) *Descriptor {
	if d.IsManifest() {
		return d.Manifest().GetBlobDescriptor(digest)
	}
	if d.IsIndex() {
		return d.Index().GetBlobDescriptor(digest)
	}
	return nil
}

func (d Artifact) MarshalJSON() ([]byte, error) {
	if d.manifest != nil {
		d.manifest.MediaType = ArtifactMimeType(d.manifest.MediaType, ociv1.MediaTypeImageManifest, legacy)
		return json.Marshal(d.manifest)
	}
	if d.index != nil {
		d.index.MediaType = ArtifactMimeType(d.index.MediaType, ociv1.MediaTypeImageIndex, legacy)
		return json.Marshal(d.index)
	}
	return []byte("null"), nil
}

func (d *Artifact) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var m helper.GenericDescriptor

	err := json.Unmarshal(data, &m)
	if err != nil {
		return err
	}

	err = m.Validate()
	if err != nil {
		return err
	}
	if m.IsManifest() {
		d.manifest = (*Manifest)(m.AsManifest())
		d.index = nil
	} else {
		d.index = (*Index)(m.AsIndex())
		d.manifest = nil
	}
	return nil
}

func Decode(data []byte) (*Artifact, error) {
	var d Artifact

	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func Encode(d *Artifact) ([]byte, error) {
	return json.Marshal(d)
}
