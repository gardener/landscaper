// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package support

import (
	"compress/gzip"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
)

type ManifestAccess struct {
	master cpi.ArtifactAccess
	artifactBase
}

var _ cpi.ManifestAccess = (*ManifestAccess)(nil)

type manifestMapper struct {
	accessobj.State
}

var _ accessobj.State = (*manifestMapper)(nil)

func (m *manifestMapper) GetState() interface{} {
	return m.State.GetState().(*artdesc.Artifact).Manifest()
}

func (m *manifestMapper) GetOriginalState() interface{} {
	return m.State.GetOriginalState().(*artdesc.Artifact).Manifest()
}

func NewManifestForArtifact(master cpi.ArtifactAccess, a *ArtifactAccessImpl) *ManifestAccess {
	m := &ManifestAccess{
		master:       master,
		artifactBase: newArtifactBase(a.container, &manifestMapper{a.state}),
	}
	return m
}

func (m *ManifestAccess) AddBlob(access cpi.BlobAccess) error {
	return m.master.AddBlob(access)
}

func (m *ManifestAccess) Manifest() (*artdesc.Manifest, error) {
	return m.GetDescriptor(), nil
}

func (m *ManifestAccess) Index() (*artdesc.Index, error) {
	return nil, errors.ErrInvalid()
}

func (m *ManifestAccess) Artifact() *artdesc.Artifact {
	a := artdesc.New()
	_ = a.SetManifest(m.GetDescriptor())
	return a
}

func (m *ManifestAccess) GetDescriptor() *artdesc.Manifest {
	return m.state.GetState().(*artdesc.Manifest)
}

func (m *ManifestAccess) GetBlobDescriptor(digest digest.Digest) *cpi.Descriptor {
	d := m.GetDescriptor().GetBlobDescriptor(digest)
	/*
		if d != nil {
			d = m.container.GetBlobDescriptor(digest)
		}
	*/
	return d
}

func (m *ManifestAccess) GetConfigBlob() (cpi.BlobAccess, error) {
	if m.GetDescriptor().Config.Digest == "" {
		return nil, nil
	}
	return m.master.GetBlob(m.GetDescriptor().Config.Digest)
}

func (m *ManifestAccess) GetBlob(digest digest.Digest) (cpi.BlobAccess, error) {
	return m.master.GetBlob(digest)
}

func (m *ManifestAccess) SetConfigBlob(blob cpi.BlobAccess, d *artdesc.Descriptor) error {
	if d == nil {
		d = artdesc.DefaultBlobDescriptor(blob)
	}
	err := m.master.AddBlob(blob)
	if err != nil {
		return err
	}
	m.GetDescriptor().Config = *d
	return nil
}

func (m *ManifestAccess) AddLayer(blob cpi.BlobAccess, d *artdesc.Descriptor) (int, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if d == nil {
		d = &artdesc.Descriptor{}
	}
	d.Digest = blob.Digest()
	d.Size = blob.Size()
	if d.MediaType == "" {
		d.MediaType = blob.MimeType()
		if d.MediaType == "" {
			d.MediaType = artdesc.MediaTypeImageLayer
			r, err := blob.Reader()
			if err != nil {
				return -1, err
			}
			defer r.Close()
			zr, err := gzip.NewReader(r)
			if err == nil {
				err = zr.Close()
				if err == nil {
					d.MediaType = artdesc.MediaTypeImageLayerGzip
				}
			}
		}
	}

	err := m.container.AddBlob(blob)
	if err != nil {
		return -1, err
	}

	manifest := m.GetDescriptor()
	manifest.Layers = append(manifest.Layers, *d)
	return len(manifest.Layers) - 1, nil
}
