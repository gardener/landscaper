// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"fmt"
	"sync"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/errors"
)

type artifactBase struct {
	lock      sync.RWMutex
	container ArtifactSetContainer
	provider  ArtifactProvider
	state     accessobj.State
}

func (a *artifactBase) IsClosed() bool {
	return a.provider.IsClosed()
}

func (a *artifactBase) IsReadOnly() bool {
	return a.provider.IsReadOnly()
}

func (a *artifactBase) IsIndex() bool {
	d := a.state.GetState().(*artdesc.Artifact)
	return d.IsIndex()
}

func (a *artifactBase) IsManifest() bool {
	d := a.state.GetState().(*artdesc.Artifact)
	return d.IsManifest()
}

func (a *artifactBase) blob() (accessio.BlobAccess, error) {
	return a.state.GetBlob()
}

func (a *artifactBase) addBlob(access BlobAccess) error {
	return a.provider.AddBlob(access)
}

func (a *artifactBase) getArtifact(digest digest.Digest) (ArtifactAccess, error) {
	return a.provider.GetArtifact(digest)
}

func (a *artifactBase) Close() error {
	return a.provider.Close()
}

func (a *artifactBase) newArtifact(art ...*artdesc.Artifact) (ArtifactAccess, error) {
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
