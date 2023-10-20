// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package artdesc

import (
	"encoding/json"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/open-component-model/ocm/pkg/common/accessio"
)

type Manifest ociv1.Manifest

var _ BlobDescriptorSource = (*Manifest)(nil)

func NewManifest() *Manifest {
	return &Manifest{
		Versioned:   specs.Versioned{SchemeVersion},
		MediaType:   MediaTypeImageManifest,
		Layers:      nil,
		Annotations: nil,
	}
}

func (i *Manifest) IsValid() bool {
	return true
}

func (m *Manifest) GetBlobDescriptor(digest digest.Digest) *Descriptor {
	if m.Config.Digest == digest {
		d := m.Config
		return &d
	}
	for _, l := range m.Layers {
		if l.Digest == digest {
			return &l
		}
	}
	return nil
}

func (m *Manifest) MimeType() string {
	return ArtifactMimeType(m.MediaType, MediaTypeImageManifest, legacy)
}

func (m *Manifest) ToBlobAccess() (accessio.BlobAccess, error) {
	m.MediaType = m.MimeType()
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return accessio.BlobAccessForData(m.MediaType, data), nil
}

func (m *Manifest) SetAnnotation(name, value string) {
	if m.Annotations == nil {
		m.Annotations = map[string]string{}
	}
	m.Annotations[name] = value
}

func (m *Manifest) DeleteAnnotation(name string) {
	if m.Annotations == nil {
		return
	}
	delete(m.Annotations, name)
	if len(m.Annotations) == 0 {
		m.Annotations = nil
	}
}

////////////////////////////////////////////////////////////////////////////////

func DecodeManifest(data []byte) (*Manifest, error) {
	var d Manifest

	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func EncodeManifest(d *Manifest) ([]byte, error) {
	return json.Marshal(d)
}
