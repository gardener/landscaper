// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
)

type Repository interface {
	GetSpecification() RepositorySpec
	NamespaceLister() NamespaceLister

	ExistsArtifact(name string, ref string) (bool, error)
	LookupArtifact(name string, ref string) (ArtifactAccess, error)
	LookupNamespace(name string) (NamespaceAccess, error)
	Close() error
}

type RepositorySource interface {
	GetRepository() Repository
}

type (
	BlobAccess = accessio.BlobAccess
	DataAccess = accessio.DataAccess
)

type BlobSource interface {
	GetBlobData(digest digest.Digest) (int64, DataAccess, error)
}

type BlobSink interface {
	AddBlob(BlobAccess) error
}

type ArtifactSink interface {
	AddBlob(BlobAccess) error
	AddArtifact(a Artifact, tags ...string) (BlobAccess, error)
	AddTags(digest digest.Digest, tags ...string) error
}

type ArtifactSource interface {
	GetArtifact(version string) (ArtifactAccess, error)
	GetBlobData(digest digest.Digest) (int64, DataAccess, error)
}

type NamespaceAccess interface {
	ArtifactSource
	ArtifactSink

	GetNamespace() string
	ListTags() ([]string, error)

	NewArtifact(...*artdesc.Artifact) (ArtifactAccess, error)

	Close() error
}

type Artifact interface {
	IsManifest() bool
	IsIndex() bool

	Digest() digest.Digest
	Blob() (BlobAccess, error)
	Artifact() *artdesc.Artifact
	Manifest() (*artdesc.Manifest, error)
	Index() (*artdesc.Index, error)
}

type ArtifactAccess interface {
	Artifact
	BlobSource
	BlobSink

	GetDescriptor() *artdesc.Artifact
	ManifestAccess() ManifestAccess
	IndexAccess() IndexAccess
	GetBlob(digest digest.Digest) (BlobAccess, error)

	GetArtifact(digest digest.Digest) (ArtifactAccess, error)
	AddBlob(BlobAccess) error

	AddArtifact(Artifact, *artdesc.Platform) (BlobAccess, error)
	AddLayer(BlobAccess, *artdesc.Descriptor) (int, error)

	Close() error
}

type ManifestAccess interface {
	Artifact

	GetDescriptor() *artdesc.Manifest
	GetBlobDescriptor(digest digest.Digest) *artdesc.Descriptor
	GetConfigBlob() (BlobAccess, error)
	GetBlob(digest digest.Digest) (BlobAccess, error)

	AddBlob(BlobAccess) error
	AddLayer(BlobAccess, *artdesc.Descriptor) (int, error)
	SetConfigBlob(blob BlobAccess, d *artdesc.Descriptor) error
}

type IndexAccess interface {
	Artifact

	GetDescriptor() *artdesc.Index
	GetBlobDescriptor(digest digest.Digest) *artdesc.Descriptor
	GetBlob(digest digest.Digest) (BlobAccess, error)

	GetArtifact(digest digest.Digest) (ArtifactAccess, error)
	/*
		GetIndex(digest digest.Digest) (IndexAccess, error)
		GetManifest(digest digest.Digest) (ManifestAccess, error)
	*/

	AddBlob(BlobAccess) error
	AddArtifact(Artifact, *artdesc.Platform) (BlobAccess, error)
}

// NamespaceLister provides the optional repository list functionality of
// a repository.
type NamespaceLister interface {
	// NumNamespaces returns the number of namespaces found for a prefix
	// If the given prefix does not end with a /, a repository with the
	// prefix name is included
	NumNamespaces(prefix string) (int, error)

	// GetNamespaces returns the name of namespaces found for a prefix
	// If the given prefix does not end with a /, a repository with the
	// prefix name is included
	GetNamespaces(prefix string, closure bool) ([]string, error)
}
