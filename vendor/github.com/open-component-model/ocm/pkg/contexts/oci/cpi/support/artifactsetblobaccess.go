// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package support

import (
	"sync"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
)

////////////////////////////////////////////////////////////////////////////////

type ArtifactSetBlobAccess struct {
	base NamespaceAccessImpl

	lock      sync.RWMutex
	blobinfos map[digest.Digest]*cpi.Descriptor
}

func NewArtifactSetBlobAccess(container NamespaceAccessImpl) *ArtifactSetBlobAccess {
	s := &ArtifactSetBlobAccess{
		base:      container,
		blobinfos: map[digest.Digest]*cpi.Descriptor{},
	}
	return s
}

func (a *ArtifactSetBlobAccess) IsReadOnly() bool {
	return a.base.IsReadOnly()
}

////////////////////////////////////////////////////////////////////////////////
// methods for BlobHandler

func (a *ArtifactSetBlobAccess) GetBlobData(digest digest.Digest) (int64, cpi.DataAccess, error) {
	return a.base.GetBlobData(digest)
}

func (a *ArtifactSetBlobAccess) GetBlob(digest digest.Digest) (cpi.BlobAccess, error) {
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

func (a *ArtifactSetBlobAccess) GetBlobDescriptor(digest digest.Digest) *cpi.Descriptor {
	a.lock.RLock()
	defer a.lock.RUnlock()

	d := a.blobinfos[digest]
	/*
		if d == nil {
			d = a.base.GetBlobDescriptor(digest)
		}
	*/
	return d
}

func (a *ArtifactSetBlobAccess) AddArtifact(artifact cpi.Artifact, tags ...string) (access accessio.BlobAccess, err error) {
	return a.base.AddArtifact(artifact, tags...)
}

func (a *ArtifactSetBlobAccess) AddBlob(blob cpi.BlobAccess) error {
	a.lock.RLock()
	defer a.lock.RUnlock()
	err := a.base.AddBlob(blob)
	if err != nil {
		return err
	}
	a.blobinfos[blob.Digest()] = artdesc.DefaultBlobDescriptor(blob)
	return nil
}
