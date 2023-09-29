// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package support

import (
	"compress/gzip"
	"fmt"
	"io"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/oci/internal"
	"github.com/open-component-model/ocm/pkg/errors"
)

var ErrNoIndex = errors.New("manifest does not support access to subsequent artifacts")

type ArtifactAccessImpl struct {
	cpi.ArtifactAccessImplBase
	artifactBase
}

var _ cpi.ArtifactAccessImpl = (*ArtifactAccessImpl)(nil)

func NewArtifactForBlob(container NamespaceAccessImpl, blob accessio.BlobAccess, closer ...io.Closer) (cpi.ArtifactAccess, error) {
	mode := accessobj.ACC_WRITABLE
	if container.IsReadOnly() {
		mode = accessobj.ACC_READONLY
	}
	state, err := accessobj.NewBlobStateForBlob(mode, blob, cpi.NewArtifactStateHandler())
	if err != nil {
		return nil, err
	}

	return newArtifact(container, state, closer...)
}

func NewArtifact(container NamespaceAccessImpl, defs ...*artdesc.Artifact) (cpi.ArtifactAccess, error) {
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
		return nil, fmt.Errorf("failed to fetch new blob state: %w", err)
	}
	return newArtifact(container, state)
}

func newArtifact(container NamespaceAccessImpl, state accessobj.State, closer ...io.Closer) (cpi.ArtifactAccess, error) {
	base, err := cpi.NewArtifactAccessImplBase(container, closer...)
	if err != nil {
		return nil, err
	}
	impl := &ArtifactAccessImpl{
		ArtifactAccessImplBase: *base,
		artifactBase:           newArtifactBase(container, state),
	}
	return cpi.NewArtifactAccess(impl), nil
}

func (a *ArtifactAccessImpl) AddBlob(access cpi.BlobAccess) error {
	return a.container.AddBlob(access)
}

func (a *ArtifactAccessImpl) NewArtifact(art ...*artdesc.Artifact) (cpi.ArtifactAccess, error) {
	if !a.IsIndex() {
		return nil, ErrNoIndex
	}
	if a.IsReadOnly() {
		return nil, accessio.ErrReadOnly
	}
	return NewArtifact(a.container, art...)
}

////////////////////////////////////////////////////////////////////////////////

func (a *ArtifactAccessImpl) Artifact() *artdesc.Artifact {
	return a.GetDescriptor()
}

func (a *ArtifactAccessImpl) GetDescriptor() *artdesc.Artifact {
	d := a.state.GetState().(*artdesc.Artifact)
	if d.IsValid() {
		return d
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// from artdesc.Artifact

func (a *ArtifactAccessImpl) GetBlobDescriptor(digest digest.Digest) *cpi.Descriptor {
	d := a.GetDescriptor().GetBlobDescriptor(digest)
	/*
		if d == nil {
			d = a.container.GetBlobDescriptor(digest)
		}
	*/
	return d
}

func (a *ArtifactAccessImpl) Index() (*artdesc.Index, error) {
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

func (a *ArtifactAccessImpl) Manifest() (*artdesc.Manifest, error) {
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

func (a *ArtifactAccessImpl) ManifestAccess(v cpi.ArtifactAccess) internal.ManifestAccess {
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
	return NewManifestForArtifact(v, a)
}

func (a *ArtifactAccessImpl) IndexAccess(v cpi.ArtifactAccess) internal.IndexAccess {
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
	return NewIndexForArtifact(v, a)
}

func (a *ArtifactAccessImpl) GetArtifact(digest digest.Digest) (cpi.ArtifactAccess, error) {
	if !a.IsIndex() {
		return nil, ErrNoIndex
	}
	return a.container.GetArtifact("@" + digest.String())
}

func (a *ArtifactAccessImpl) GetBlobData(digest digest.Digest) (int64, cpi.DataAccess, error) {
	return a.container.GetBlobData(digest)
}

func (a *ArtifactAccessImpl) GetBlob(digest digest.Digest) (cpi.BlobAccess, error) {
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

func (a *ArtifactAccessImpl) AddArtifact(art cpi.Artifact, platform *artdesc.Platform) (cpi.BlobAccess, error) {
	if a.IsReadOnly() {
		return nil, accessio.ErrReadOnly
	}
	d, err := a.Index()
	if err != nil {
		return nil, err
	}

	a.lock.Lock()
	defer a.lock.Unlock()

	blob, err := a.container.AddArtifact(art)
	if err != nil {
		return nil, err
	}
	d.Manifests = append(d.Manifests, cpi.Descriptor{
		MediaType:   blob.MimeType(),
		Digest:      blob.Digest(),
		Size:        blob.Size(),
		URLs:        nil,
		Annotations: nil,
		Platform:    platform,
	})
	return blob, nil
}

func (a *ArtifactAccessImpl) AddLayer(blob cpi.BlobAccess, d *cpi.Descriptor) (int, error) {
	if a.IsReadOnly() {
		return -1, accessio.ErrReadOnly
	}
	m, err := a.Manifest()
	if err != nil {
		return -1, err
	}

	a.lock.Lock()
	defer a.lock.Unlock()
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

	err = a.container.AddBlob(blob)
	if err != nil {
		return -1, err
	}

	m.Layers = append(m.Layers, *d)
	return len(m.Layers) - 1, nil
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
