// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package support

import (
	"fmt"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/oci/internal"
	"github.com/open-component-model/ocm/pkg/errors"
)

var ErrNoIndex = errors.New("manifest does not support access to subsequent artifacts")

type ArtifactImpl struct {
	*artifactBase
}

var _ cpi.ArtifactAccess = (*ArtifactImpl)(nil)

func NewArtifactForBlob(container ArtifactSetContainerImpl, blob accessio.BlobAccess) (cpi.ArtifactAccess, error) {
	mode := accessobj.ACC_WRITABLE
	if container.IsReadOnly() {
		mode = accessobj.ACC_READONLY
	}
	state, err := accessobj.NewBlobStateForBlob(mode, blob, cpi.NewArtifactStateHandler())
	if err != nil {
		return nil, err
	}

	return newArtifactImpl(container, state)
}

func NewArtifact(container ArtifactSetContainerImpl, defs ...*artdesc.Artifact) (cpi.ArtifactAccess, error) {
	var def *artdesc.Artifact
	if len(defs) != 0 && defs[0] != nil {
		def = defs[0]
	}
	mode := accessobj.ACC_WRITABLE
	if container.IsReadOnly() {
		mode = accessobj.ACC_READONLY
	}
	state, err := accessobj.NewBlobStateForObject(mode, def, cpi.NewArtifactStateHandler())
	if err != nil {
		panic("oops: " + err.Error())
	}
	return newArtifactImpl(container, state)
}

func newArtifactImpl(container ArtifactSetContainerImpl, state accessobj.State) (cpi.ArtifactAccess, error) {
	v, err := container.View()
	if err != nil {
		return nil, err
	}
	a := &ArtifactImpl{
		artifactBase: newArtifactBase(v, container, state),
	}
	return a, nil
}

func (a *ArtifactImpl) Close() error {
	return a.view.Close()
}

////////////////////////////////////////////////////////////////////////////////
// forward

func (a *ArtifactImpl) AddBlob(access cpi.BlobAccess) error {
	return a.addBlob(access)
}

func (a *ArtifactImpl) NewArtifact(art ...*artdesc.Artifact) (cpi.ArtifactAccess, error) {
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

func (a *ArtifactImpl) GetBlobDescriptor(digest digest.Digest) *cpi.Descriptor {
	d := a.GetDescriptor().GetBlobDescriptor(digest)
	if d != nil {
		return d
	}
	return a.container.GetBlobDescriptor(digest)
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
	d := a.state.GetState().(*artdesc.Artifact)
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

func (a *ArtifactImpl) GetArtifact(digest digest.Digest) (cpi.ArtifactAccess, error) {
	if !a.IsIndex() {
		return nil, ErrNoIndex
	}
	return a.container.GetArtifact("@" + digest.String())
}

func (a *ArtifactImpl) GetBlobData(digest digest.Digest) (int64, cpi.DataAccess, error) {
	return a.container.GetBlobData(digest)
}

func (a *ArtifactImpl) GetBlob(digest digest.Digest) (cpi.BlobAccess, error) {
	d := a.GetBlobDescriptor(digest)
	if d != nil {
		size, data, err := a.container.GetBlobData(digest)
		if err != nil {
			return nil, err
		}
		err = AdjustSize(d, size)
		if err != nil {
			return nil, err
		}
		return accessio.BlobAccessForDataAccess(d.Digest, d.Size, d.MediaType, data), nil
	}
	return nil, cpi.ErrBlobNotFound(digest)
}

func (a *ArtifactImpl) AddArtifact(art cpi.Artifact, platform *artdesc.Platform) (cpi.BlobAccess, error) {
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

func (a *ArtifactImpl) AddLayer(blob cpi.BlobAccess, d *cpi.Descriptor) (int, error) {
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
