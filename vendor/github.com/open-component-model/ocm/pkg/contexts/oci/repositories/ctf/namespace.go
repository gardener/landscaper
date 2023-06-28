// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ctf

import (
	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi/support"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/ctf/index"
	"github.com/open-component-model/ocm/pkg/errors"
)

func NewNamespace(repo *RepositoryImpl, name string) (cpi.NamespaceAccess, error) {
	return support.NewNamespaceAccess(name, newNamespaceContainer(repo), repo, "CTF namespace")
}

type namespaceContainer struct {
	impl support.NamespaceAccessImpl
	repo *RepositoryImpl
}

var _ support.NamespaceContainer = (*namespaceContainer)(nil)

func newNamespaceContainer(repo *RepositoryImpl) support.NamespaceContainer {
	return &namespaceContainer{
		repo: repo,
	}
}

func (n *namespaceContainer) SetImplementation(impl support.NamespaceAccessImpl) {
	n.impl = impl
}

func (n *namespaceContainer) IsReadOnly() bool {
	return n.repo.IsReadOnly()
}

func (n *namespaceContainer) Close() error {
	return nil
}

func (n *namespaceContainer) GetBlobDescriptor(digest digest.Digest) *cpi.Descriptor {
	return nil
}

func (n *namespaceContainer) ListTags() ([]string, error) {
	return n.repo.getIndex().GetTags(n.impl.GetNamespace()), nil // return digests as tags, also
}

func (n *namespaceContainer) GetBlobData(digest digest.Digest) (int64, cpi.DataAccess, error) {
	return n.repo.base.GetBlobData(digest)
}

func (n *namespaceContainer) AddBlob(blob cpi.BlobAccess) error {
	n.repo.base.Lock()
	defer n.repo.base.Unlock()

	return n.repo.base.AddBlob(blob)
}

func (n *namespaceContainer) GetArtifact(i support.NamespaceAccessImpl, vers string) (cpi.ArtifactAccess, error) {
	meta := n.repo.getIndex().GetArtifactInfo(n.impl.GetNamespace(), vers)
	if meta == nil {
		return nil, errors.ErrNotFound(cpi.KIND_OCIARTIFACT, vers, n.impl.GetNamespace())
	}
	return n.repo.base.GetArtifact(i, meta.Digest)
}

func (n *namespaceContainer) HasArtifact(vers string) (bool, error) {
	meta := n.repo.getIndex().GetArtifactInfo(n.impl.GetNamespace(), vers)
	return meta != nil, nil
}

func (n *namespaceContainer) AddArtifact(artifact cpi.Artifact, tags ...string) (access accessio.BlobAccess, err error) {
	n.repo.base.Lock()
	defer n.repo.base.Unlock()

	blob, err := n.repo.base.AddArtifactBlob(artifact)
	if err != nil {
		return nil, err
	}
	n.repo.getIndex().AddArtifactInfo(&index.ArtifactMeta{
		Repository: n.impl.GetNamespace(),
		Tag:        "",
		Digest:     blob.Digest(),
	})
	return blob, n.AddTags(blob.Digest(), tags...)
}

func (n *namespaceContainer) AddTags(digest digest.Digest, tags ...string) error {
	return n.repo.getIndex().AddTagsFor(n.impl.GetNamespace(), digest, tags...)
}

func (n *namespaceContainer) NewArtifact(i support.NamespaceAccessImpl, art ...*artdesc.Artifact) (cpi.ArtifactAccess, error) {
	if n.IsReadOnly() {
		return nil, accessio.ErrReadOnly
	}
	return support.NewArtifact(i, art...)
}
