// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package support

import (
	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
)

// ArtifactSetContainer is the interface used by subsequent access objects
// to access the base implementation.
type ArtifactSetContainer interface {
	IsReadOnly() bool
	IsClosed() bool

	Close() error

	GetBlobDescriptor(digest digest.Digest) *cpi.Descriptor
	GetBlobData(digest digest.Digest) (int64, cpi.DataAccess, error)
	AddBlob(blob cpi.BlobAccess) error

	GetArtifact(vers string) (cpi.ArtifactAccess, error)
	AddArtifact(artifact cpi.Artifact, tags ...string) (access accessio.BlobAccess, err error)
}

////////////////////////////////////////////////////////////////////////////////

// ArtifactSetContainerInt is the implementation interface for a provider.
type ArtifactSetContainerInt ArtifactSetContainer

type artifactSetContainerImpl struct {
	refs accessio.ReferencableCloser
	ArtifactSetContainerInt
}

type ArtifactSetContainerImpl interface {
	ArtifactSetContainer
	View(main ...bool) (ArtifactSetContainer, error)
}

func NewArtifactSetContainer(c ArtifactSetContainerInt) (ArtifactSetContainer, ArtifactSetContainerImpl) {
	i := &artifactSetContainerImpl{
		refs:                    accessio.NewRefCloser(c, true),
		ArtifactSetContainerInt: c,
	}
	v, _ := i.View(true)
	return v, i
}

func (i *artifactSetContainerImpl) View(main ...bool) (ArtifactSetContainer, error) {
	v, err := i.refs.View(main...)
	if err != nil {
		return nil, err
	}
	return &artifactSetContainerView{
		view:                     v,
		ArtifactSetContainerImpl: i,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////

type artifactSetContainerView struct {
	view accessio.CloserView
	ArtifactSetContainerImpl
}

func (v *artifactSetContainerView) IsClosed() bool {
	return v.view.IsClosed()
}

func (v *artifactSetContainerView) Close() error {
	return v.view.Close()
}
