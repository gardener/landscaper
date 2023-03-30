// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"fmt"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/internal"
	"github.com/open-component-model/ocm/pkg/errors"
)

var ErrNoIndex = errors.New("manifest does not support access to subsequent artifacts")

type ArtifactImpl struct {
	artifactBase
}

var _ ArtifactAccess = (*ArtifactImpl)(nil)

func NewArtifactForProviderBlob(access ArtifactSetContainer, p ArtifactProvider, blob accessio.BlobAccess) (*ArtifactImpl, error) {
	mode := accessobj.ACC_WRITABLE
	if access.IsReadOnly() {
		mode = accessobj.ACC_READONLY
	}
	state, err := accessobj.NewBlobStateForBlob(mode, blob, NewArtifactStateHandler())
	if err != nil {
		return nil, err
	}
	a := &ArtifactImpl{
		artifactBase: artifactBase{
			container: access,
			state:     state,
			provider:  p,
		},
	}
	return a, nil
}

func NewArtifactForBlob(access ArtifactSetContainer, blob accessio.BlobAccess) (*ArtifactImpl, error) {
	mode := accessobj.ACC_WRITABLE
	if access.IsReadOnly() {
		mode = accessobj.ACC_READONLY
	}
	state, err := accessobj.NewBlobStateForBlob(mode, blob, NewArtifactStateHandler())
	if err != nil {
		return nil, err
	}
	p, err := access.NewArtifactProvider(state)
	if err != nil {
		return nil, err
	}
	a := &ArtifactImpl{
		artifactBase: artifactBase{
			container: access,
			state:     state,
			provider:  p,
		},
	}
	return a, nil
}

func NewArtifact(access ArtifactSetContainer, defs ...*artdesc.Artifact) (ArtifactAccess, error) {
	var def *artdesc.Artifact
	if len(defs) != 0 && defs[0] != nil {
		def = defs[0]
	}
	mode := accessobj.ACC_WRITABLE
	if access.IsReadOnly() {
		mode = accessobj.ACC_READONLY
	}
	state, err := accessobj.NewBlobStateForObject(mode, def, NewArtifactStateHandler())
	if err != nil {
		panic("oops: " + err.Error())
	}

	p, err := access.NewArtifactProvider(state)
	if err != nil {
		return nil, err
	}
	a := &ArtifactImpl{
		artifactBase: artifactBase{
			container: access,
			provider:  p,
			state:     state,
		},
	}
	return a, nil
}

////////////////////////////////////////////////////////////////////////////////
// forward

func (a *ArtifactImpl) AddBlob(access BlobAccess) error {
	return a.addBlob(access)
}

func (a *ArtifactImpl) NewArtifact(art ...*artdesc.Artifact) (ArtifactAccess, error) {
	if !a.IsIndex() {
		return nil, ErrNoIndex
	}
	return a.newArtifact(art...)
}

////////////////////////////////////////////////////////////////////////////////

func (a *ArtifactImpl) Artifact() *artdesc.Artifact {
	return a.GetDescriptor()
}

func (a *ArtifactImpl) GetDescriptor() *artdesc.Artifact {
	d := a.state.GetState().(*artdesc.Artifact)
	if d.IsValid() {
		return d
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// from artdesc.Artifact

func (a *ArtifactImpl) GetBlobDescriptor(digest digest.Digest) *Descriptor {
	d := a.GetDescriptor().GetBlobDescriptor(digest)
	if d != nil {
		return d
	}
	return a.provider.GetBlobDescriptor(digest)
	// return a.container.GetBlobDescriptor(digest)
}

func (a *ArtifactImpl) Index() (*artdesc.Index, error) {
	a.lock.Lock()
	defer a.lock.Unlock()
	d, ok := a.state.GetState().(*artdesc.Artifact)
	if !ok {
		return nil, fmt.Errorf("failed to assert type %T to *artdesc.Artifact", a.state.GetState())
	}
	idx := d.Index()
	if idx == nil {
		idx = artdesc.NewIndex()
		if err := d.SetIndex(idx); err != nil {
			return nil, errors.Newf("artifact is manifest")
		}
	}
	return idx, nil
}

func (a *ArtifactImpl) Manifest() (*artdesc.Manifest, error) {
	a.lock.Lock()
	defer a.lock.Unlock()
	d, ok := a.state.GetState().(*artdesc.Artifact)
	if !ok {
		return nil, fmt.Errorf("failed to assert type %T to *artdesc.Artifact", a.state.GetState())
	}
	m := d.Manifest()
	if m == nil {
		m = artdesc.NewManifest()
		if err := d.SetManifest(m); err != nil {
			return nil, errors.Newf("artifact is index")
		}
	}
	return m, nil
}

func (a *ArtifactImpl) ManifestAccess() internal.ManifestAccess {
	a.lock.Lock()
	defer a.lock.Unlock()
	d := a.state.GetState().(*artdesc.Artifact)
	m := d.Manifest()
	if m == nil {
		m = artdesc.NewManifest()
		if err := d.SetManifest(m); err != nil {
			return nil
		}
	}
	return NewManifestForArtifact(a)
}

func (a *ArtifactImpl) IndexAccess() internal.IndexAccess {
	a.lock.Lock()
	defer a.lock.Unlock()
	d := a.state.GetState().(*artdesc.Artifact)
	i := d.Index()
	if i == nil {
		i = artdesc.NewIndex()
		if err := d.SetIndex(i); err != nil {
			return nil
		}
	}
	return NewIndexForArtifact(a)
}

func (a *ArtifactImpl) GetArtifact(digest digest.Digest) (ArtifactAccess, error) {
	if !a.IsIndex() {
		return nil, ErrNoIndex
	}
	return a.getArtifact(digest)
}

func (a *ArtifactImpl) GetBlobData(digest digest.Digest) (int64, DataAccess, error) {
	return a.provider.GetBlobData(digest)
}

func (a *ArtifactImpl) GetBlob(digest digest.Digest) (BlobAccess, error) {
	d := a.GetBlobDescriptor(digest)
	if d != nil {
		size, data, err := a.provider.GetBlobData(digest)
		if err != nil {
			return nil, err
		}
		err = AdjustSize(d, size)
		if err != nil {
			return nil, err
		}
		return accessio.BlobAccessForDataAccess(d.Digest, d.Size, d.MediaType, data), nil
	}
	return nil, ErrBlobNotFound(digest)
}

func (a *ArtifactImpl) AddArtifact(art Artifact, platform *artdesc.Platform) (accessio.BlobAccess, error) {
	if a.IsClosed() {
		return nil, accessio.ErrClosed
	}
	if a.IsReadOnly() {
		return nil, accessio.ErrReadOnly
	}
	_, err := a.Index()
	if err != nil {
		return nil, err
	}
	return NewIndexForArtifact(a).AddArtifact(art, platform)
}

func (a *ArtifactImpl) AddLayer(blob BlobAccess, d *Descriptor) (int, error) {
	if a.IsClosed() {
		return -1, accessio.ErrClosed
	}
	if a.IsReadOnly() {
		return -1, accessio.ErrReadOnly
	}
	_, err := a.Manifest()
	if err != nil {
		return -1, err
	}
	return NewManifestForArtifact(a).AddLayer(blob, d)
}

func AdjustSize(d *artdesc.Descriptor, size int64) error {
	if size != accessio.BLOB_UNKNOWN_SIZE {
		if d.Size == accessio.BLOB_UNKNOWN_SIZE {
			d.Size = size
		} else if d.Size != size {
			return errors.Newf("blob size mismatch %d != %d", size, d.Size)
		}
	}
	return nil
}
