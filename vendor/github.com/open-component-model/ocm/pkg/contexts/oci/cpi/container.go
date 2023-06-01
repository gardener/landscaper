// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
)

// ArtifactProvider manages the technical access to a dedicated artifact.
type ArtifactProvider interface {
	IsClosed() bool
	IsReadOnly() bool
	GetBlobDescriptor(digest digest.Digest) *Descriptor
	Close() error

	BlobSource
	BlobSink

	// GetArtifact is used to access nested artifacts (only)
	GetArtifact(digest digest.Digest) (ArtifactAccess, error)
	// AddArtifact is used to add nested artifacts (only)
	AddArtifact(art Artifact) (access accessio.BlobAccess, err error)
}

type NopCloserArtifactProvider struct {
	ArtifactSetContainer
}

var _ ArtifactProvider = (*NopCloserArtifactProvider)(nil)

func (p *NopCloserArtifactProvider) Close() error {
	return nil
}

func (p *NopCloserArtifactProvider) AddArtifact(art Artifact) (access accessio.BlobAccess, err error) {
	return p.ArtifactSetContainer.AddArtifact(art)
}

func (p *NopCloserArtifactProvider) GetArtifact(digest digest.Digest) (ArtifactAccess, error) {
	return p.ArtifactSetContainer.GetArtifact("@" + digest.String())
}

func NewNopCloserArtifactProvider(p ArtifactSetContainer) ArtifactProvider {
	return &NopCloserArtifactProvider{
		p,
	}
}

////////////////////////////////////////////////////////////////////////////////

// ArtifactSetContainer is the interface used by subsequent access objects
// to access the base implementation.
type ArtifactSetContainer interface {
	IsReadOnly() bool
	IsClosed() bool

	Close() error

	GetBlobDescriptor(digest digest.Digest) *Descriptor
	GetBlobData(digest digest.Digest) (int64, DataAccess, error)
	AddBlob(blob BlobAccess) error

	GetArtifact(vers string) (ArtifactAccess, error)
	AddArtifact(artifact Artifact, tags ...string) (access accessio.BlobAccess, err error)

	NewArtifactProvider(state accessobj.State) (ArtifactProvider, error)
}

////////////////////////////////////////////////////////////////////////////////

type artifactSetContainerImpl struct {
	refs accessio.ReferencableCloser
	ArtifactSetContainer
}

func NewArtifactSetContainer(c ArtifactSetContainer) ArtifactSetContainer {
	i := &artifactSetContainerImpl{
		refs:                 accessio.NewRefCloser(c, true),
		ArtifactSetContainer: c,
	}
	v, _ := i.View()
	return v
}

func (i *artifactSetContainerImpl) View() (ArtifactSetContainer, error) {
	v, err := i.refs.View()
	if err != nil {
		return nil, err
	}
	return &artifactSetContainerView{
		view:                 v,
		ArtifactSetContainer: i.ArtifactSetContainer,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////

type artifactSetContainerView struct {
	view accessio.CloserView
	ArtifactSetContainer
}

func (v *artifactSetContainerView) IsClosed() bool {
	return v.view.IsClosed()
}

func (v *artifactSetContainerView) Close() error {
	return v.view.Close()
}
