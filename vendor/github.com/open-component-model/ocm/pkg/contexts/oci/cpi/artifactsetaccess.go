// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"sync"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
)

////////////////////////////////////////////////////////////////////////////////

type ArtifactSetAccess struct {
	base ArtifactSetContainer

	lock      sync.RWMutex
	blobinfos map[digest.Digest]*Descriptor
}

func NewArtifactSetAccess(container ArtifactSetContainer) *ArtifactSetAccess {
	s := &ArtifactSetAccess{
		base:      container,
		blobinfos: map[digest.Digest]*Descriptor{},
	}
	return s
}

func (a *ArtifactSetAccess) IsReadOnly() bool {
	return a.base.IsReadOnly()
}

func (a *ArtifactSetAccess) IsClosed() bool {
	return a.base.IsClosed()
}

////////////////////////////////////////////////////////////////////////////////
// methods for BlobHandler

func (a *ArtifactSetAccess) GetBlobData(digest digest.Digest) (int64, DataAccess, error) {
	return a.base.GetBlobData(digest)
}

func (a *ArtifactSetAccess) GetBlob(digest digest.Digest) (BlobAccess, error) {
	if a.IsClosed() {
		return nil, accessio.ErrClosed
	}
	size, data, err := a.GetBlobData(digest)
	if err != nil {
		return nil, err
	}
	d := a.GetBlobDescriptor(digest)
	if d != nil {
		err = AdjustSize(d, size)
		if err != nil {
			return nil, err
		}
		return accessio.BlobAccessForDataAccess(d.Digest, d.Size, d.MediaType, data), nil
	}
	return accessio.BlobAccessForDataAccess(digest, size, "", data), nil
}

func (a *ArtifactSetAccess) GetBlobDescriptor(digest digest.Digest) *Descriptor {
	a.lock.RLock()
	defer a.lock.RUnlock()

	d := a.blobinfos[digest]
	if d == nil {
		d = a.base.GetBlobDescriptor(digest)
	}
	return d
}

func (a *ArtifactSetAccess) AddArtifact(artifact Artifact, tags ...string) (access accessio.BlobAccess, err error) {
	return a.base.AddArtifact(artifact, tags...)
}

func (a *ArtifactSetAccess) AddBlob(blob BlobAccess) error {
	a.lock.RLock()
	defer a.lock.RUnlock()
	err := a.base.AddBlob(blob)
	if err != nil {
		return err
	}
	a.blobinfos[blob.Digest()] = artdesc.DefaultBlobDescriptor(blob)
	return nil
}
