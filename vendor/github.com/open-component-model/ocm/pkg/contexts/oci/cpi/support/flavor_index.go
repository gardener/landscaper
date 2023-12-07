// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package support

import (
	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/oci/internal"
	"github.com/open-component-model/ocm/pkg/errors"
)

type IndexAccess struct {
	master cpi.ArtifactAccess
	artifactBase
}

var _ cpi.IndexAccess = (*IndexAccess)(nil)

type indexMapper struct {
	accessobj.State
}

var _ accessobj.State = (*indexMapper)(nil)

func (m *indexMapper) GetState() interface{} {
	return m.State.GetState().(*artdesc.Artifact).Index()
}

func (m *indexMapper) GetOriginalState() interface{} {
	return m.State.GetOriginalState().(*artdesc.Artifact).Index()
}

func NewIndexForArtifact(master cpi.ArtifactAccess, a *ArtifactAccessImpl) *IndexAccess {
	m := &IndexAccess{
		master: master,
		artifactBase: artifactBase{
			container: a.container,
			state:     &indexMapper{a.state},
		},
	}
	return m
}

func (i *IndexAccess) NewArtifact(art ...*artdesc.Artifact) (cpi.ArtifactAccess, error) {
	return i.master.NewArtifact(art...)
}

func (i *IndexAccess) AddBlob(blob internal.BlobAccess) error {
	return i.master.AddBlob(blob)
}

func (i *IndexAccess) Manifest() (*artdesc.Manifest, error) {
	return nil, errors.ErrInvalid()
}

func (i *IndexAccess) Index() (*artdesc.Index, error) {
	return i.GetDescriptor(), nil
}

func (i *IndexAccess) Artifact() *artdesc.Artifact {
	a := artdesc.New()
	_ = a.SetIndex(i.GetDescriptor())
	return a
}

func (i *IndexAccess) GetDescriptor() *artdesc.Index {
	return i.state.GetState().(*artdesc.Index)
}

func (i *IndexAccess) GetBlobDescriptor(digest digest.Digest) *cpi.Descriptor {
	d := i.GetDescriptor().GetBlobDescriptor(digest)
	/*
		if d == nil {
			d = i.container.GetBlobDescriptor(digest)
		}
	*/
	return d
}

func (i *IndexAccess) GetBlob(digest digest.Digest) (internal.BlobAccess, error) {
	d := i.GetBlobDescriptor(digest)
	if d != nil {
		size, data, err := i.master.GetBlobData(digest)
		if err != nil {
			return nil, err
		}
		err = AdjustSize(d, size)
		if err != nil {
			return nil, err
		}
		return blobaccess.ForDataAccess(d.Digest, d.Size, d.MediaType, data), nil
	}
	return nil, cpi.ErrBlobNotFound(digest)
}

func (i *IndexAccess) GetArtifact(digest digest.Digest) (internal.ArtifactAccess, error) {
	for _, d := range i.GetDescriptor().Manifests {
		if d.Digest == digest {
			return i.master.GetArtifact(digest)
		}
	}
	return nil, errors.ErrNotFound(cpi.KIND_OCIARTIFACT, digest.String())
}

func (a *IndexAccess) AddArtifact(art cpi.Artifact, platform *artdesc.Platform) (access blobaccess.BlobAccess, err error) {
	return a.master.AddArtifact(art, platform)
}
