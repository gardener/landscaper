// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package artdesc

import (
	"encoding/json"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/open-component-model/ocm/pkg/blobaccess"
)

type Index ociv1.Index

var _ BlobDescriptorSource = (*Index)(nil)

func NewIndex() *Index {
	return &Index{
		Versioned:   specs.Versioned{SchemeVersion},
		MediaType:   MediaTypeImageIndex,
		Manifests:   nil,
		Annotations: nil,
	}
}

func (i *Index) IsValid() bool {
	return true
}

func (i *Index) GetBlobDescriptor(digest digest.Digest) *Descriptor {
	for _, m := range i.Manifests {
		if m.Digest == digest {
			return &m
		}
	}
	return nil
}

func (i *Index) MimeType() string {
	return ArtifactMimeType(i.MediaType, MediaTypeImageIndex, legacy)
}

func (i *Index) SetAnnotation(name, value string) {
	if i.Annotations == nil {
		i.Annotations = map[string]string{}
	}
	i.Annotations[name] = value
}

func (i *Index) DeleteAnnotation(name string) {
	if i.Annotations == nil {
		return
	}
	delete(i.Annotations, name)
	if len(i.Annotations) == 0 {
		i.Annotations = nil
	}
}

func (i *Index) ToBlobAccess() (blobaccess.BlobAccess, error) {
	i.MediaType = i.MimeType()
	data, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	return blobaccess.ForData(i.MediaType, data), nil
}

func (i *Index) AddManifest(d *Descriptor) {
	i.Manifests = append(i.Manifests, *d)
}

////////////////////////////////////////////////////////////////////////////////

func DecodeIndex(data []byte) (*Index, error) {
	var d Index

	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func EncodeIndex(d *Index) ([]byte, error) {
	return json.Marshal(d)
}
