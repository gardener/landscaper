// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package support

import (
	"fmt"
	"sync"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
)

type artifactBase struct {
	lock      sync.RWMutex
	view      ArtifactSetContainer
	container ArtifactSetContainerImpl
	state     accessobj.State
}

func newArtifactBase(view ArtifactSetContainer, container ArtifactSetContainerImpl, state accessobj.State) *artifactBase {
	return &artifactBase{
		view:      view,
		container: container,
		state:     state,
	}
}

func (a *artifactBase) IsClosed() bool {
	return a.view.IsClosed()
}

func (a *artifactBase) IsReadOnly() bool {
	return a.container.IsReadOnly()
}

func (a *artifactBase) IsIndex() bool {
	d := a.state.GetState().(*artdesc.Artifact)
	return d.IsIndex()
}

func (a *artifactBase) IsManifest() bool {
	d := a.state.GetState().(*artdesc.Artifact)
	return d.IsManifest()
}

func (a *artifactBase) blob() (cpi.BlobAccess, error) {
	return a.state.GetBlob()
}

func (a *artifactBase) addBlob(access cpi.BlobAccess) error {
	return a.container.AddBlob(access)
}

func (a *artifactBase) newArtifact(art ...*artdesc.Artifact) (cpi.ArtifactAccess, error) {
	if a.IsClosed() {
		return nil, accessio.ErrClosed
	}
	if a.IsReadOnly() {
		return nil, accessio.ErrReadOnly
	}
	return NewArtifact(a.container, art...)
}

func (a *artifactBase) Blob() (accessio.BlobAccess, error) {
	d, ok := a.state.GetState().(artdesc.BlobDescriptorSource)
	if !ok {
		return nil, fmt.Errorf("failed to assert type %T to artdesc.BlobDescriptorSource", a.state.GetState())
	}
	if !d.IsValid() {
		return nil, errors.ErrUnknown("artifact type")
	}
	blob, err := a.blob()
	if err != nil {
		return nil, err
	}
	return accessio.BlobWithMimeType(d.MimeType(), blob), nil
}

func (a *artifactBase) Digest() digest.Digest {
	d := a.state.GetState().(artdesc.BlobDescriptorSource)
	if !d.IsValid() {
		return ""
	}
	blob, err := a.blob()
	if err != nil {
		return ""
	}
	return blob.Digest()
}
