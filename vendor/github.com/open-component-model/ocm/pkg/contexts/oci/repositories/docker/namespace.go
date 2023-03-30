// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"fmt"
	"strings"
	"sync"

	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/types"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/mandelsoft/logging"
	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
)

type NamespaceContainer struct {
	lock      sync.RWMutex
	repo      *Repository
	namespace string
	cache     accessio.BlobCache
	log       logging.Logger
}

var (
	_ cpi.ArtifactSetContainer = (*NamespaceContainer)(nil)
	_ cpi.NamespaceAccess      = (*Namespace)(nil)
)

func NewNamespace(repo *Repository, name string) (*Namespace, error) {
	cache, err := accessio.NewCascadedBlobCache(nil)
	if err != nil {
		return nil, err
	}
	n := &Namespace{
		access: &NamespaceContainer{
			repo:      repo,
			namespace: name,
			cache:     cache,
			log:       repo.ctx.Logger(),
		},
	}
	return n, nil
}

func (n *NamespaceContainer) GetNamepace() string {
	return n.namespace
}

func (n *NamespaceContainer) IsReadOnly() bool {
	return n.repo.IsReadOnly()
}

func (n *NamespaceContainer) IsClosed() bool {
	n.lock.RLock()
	defer n.lock.RUnlock()
	return n.cache == nil
}

func (n *NamespaceContainer) Close() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.cache != nil {
		err := n.cache.Unref()
		n.cache = nil
		if err != nil {
			return fmt.Errorf("failed to unref: %w", err)
		}
	}
	return nil
}

func (n *NamespaceContainer) GetBlobDescriptor(digest digest.Digest) *cpi.Descriptor {
	return nil
}

func (n *NamespaceContainer) ListTags() ([]string, error) {
	opts := dockertypes.ImageListOptions{}
	list, err := n.repo.client.ImageList(dummyContext, opts)
	if err != nil {
		return nil, err
	}
	var result []string
	if n.namespace == "" {
		for _, e := range list {
			// ID is always the config digest
			// filter images without a repo tag for empty namespace
			if len(e.RepoTags) == 0 {
				d, err := digest.Parse(e.ID)
				if err == nil {
					result = append(result, d.String()[:12])
				}
			}
		}
	} else {
		prefix := n.namespace + ":"
		for _, e := range list {
			for _, t := range e.RepoTags {
				if strings.HasPrefix(t, prefix) {
					result = append(result, t[len(prefix):])
				}
			}
		}
	}
	return result, nil
}

func (n *NamespaceContainer) GetBlobData(digest digest.Digest) (int64, cpi.DataAccess, error) {
	return n.cache.GetBlobData(digest)
}

func (n *NamespaceContainer) AddBlob(blob cpi.BlobAccess) error {
	if _, _, err := n.cache.AddBlob(blob); err != nil {
		return fmt.Errorf("failed to add blob to cache: %w", err)
	}

	return nil
}

func (n *NamespaceContainer) GetArtifact(vers string) (cpi.ArtifactAccess, error) {
	ref, err := ParseRef(n.namespace, vers)
	if err != nil {
		return nil, err
	}
	src, err := ref.NewImageSource(dummyContext, n.repo.sysctx)
	if err != nil {
		return nil, err
	}

	opts := types.ManifestUpdateOptions{
		ManifestMIMEType: artdesc.MediaTypeImageManifest,
	}
	un := image.UnparsedInstance(src, nil)
	img, err := image.FromUnparsedImage(dummyContext, n.repo.sysctx, un)
	if err != nil {
		src.Close()
		return nil, err
	}

	img, err = img.UpdatedImage(dummyContext, opts)
	if err != nil {
		src.Close()
		return nil, err
	}

	data, mime, err := img.Manifest(dummyContext)
	if err != nil {
		src.Close()
		return nil, err
	}

	cache, err := accessio.NewCascadedBlobCacheForSource(n.cache, newDockerSource(img, src))
	if err != nil {
		return nil, err
	}
	p := &daemonArtifactProvider{
		namespace: n,
		cache:     cache,
	}
	return cpi.NewArtifactForProviderBlob(n, p, accessio.BlobAccessForData(mime, data))
}

func (n *NamespaceContainer) AddArtifact(artifact cpi.Artifact, tags ...string) (access accessio.BlobAccess, err error) {
	tag := "latest"
	if len(tags) > 0 {
		tag = tags[0]
	}
	ref, err := ParseRef(n.namespace, tag)
	if err != nil {
		return nil, err
	}
	dst, err := ref.NewImageDestination(dummyContext, nil)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	blob, err := Convert(artifact, n.cache, dst)
	if err != nil {
		return nil, err
	}
	err = dst.Commit(dummyContext, nil)
	if err != nil {
		return nil, err
	}

	return blob, nil
}

func (n *NamespaceContainer) AddTags(digest digest.Digest, tags ...string) error {
	if ok, _ := artdesc.IsDigest(digest.String()); ok {
		return errors.ErrNotSupported("image access by digest")
	}

	src := n.namespace + ":" + digest.String()

	if pattern.MatchString(digest.String()) {
		// this definitely no digest, but the library expects it this way
		src = digest.String()
	}

	for _, tag := range tags {
		err := n.repo.client.ImageTag(dummyContext, src, n.namespace+":"+tag)
		if err != nil {
			return fmt.Errorf("failed to add image tag: %w", err)
		}
	}

	return nil
}

func (n *NamespaceContainer) NewArtifactProvider(state accessobj.State) (cpi.ArtifactProvider, error) {
	return nil, nil
}

////////////////////////////////////////////////////////////////////////////////

type Namespace struct {
	access *NamespaceContainer
}

func (n *Namespace) Close() error {
	return n.access.Close()
}

func (n *Namespace) GetRepository() cpi.Repository {
	return n.access.repo
}

func (n *Namespace) GetNamespace() string {
	return n.access.GetNamepace()
}

func (n *Namespace) ListTags() ([]string, error) {
	return n.access.ListTags()
}

func (n *Namespace) NewArtifact(art ...*artdesc.Artifact) (cpi.ArtifactAccess, error) {
	if n.access.IsReadOnly() {
		return nil, accessio.ErrReadOnly
	}
	var m *artdesc.Artifact
	if len(art) == 0 {
		m = artdesc.NewManifestArtifact()
	} else {
		if !art[0].IsManifest() {
			err := m.SetManifest(artdesc.NewManifest())
			if err != nil {
				return nil, err
			}
		}
		m = art[0]
	}
	return cpi.NewArtifact(n.access, m)
}

func (n *Namespace) GetBlobData(digest digest.Digest) (int64, cpi.DataAccess, error) {
	return n.access.GetBlobData(digest)
}

func (n *Namespace) GetArtifact(vers string) (cpi.ArtifactAccess, error) {
	return n.access.GetArtifact(vers)
}

func (n *Namespace) AddArtifact(artifact cpi.Artifact, tags ...string) (accessio.BlobAccess, error) {
	return n.access.AddArtifact(artifact, tags...)
}

func (n *Namespace) AddTags(digest digest.Digest, tags ...string) error {
	return n.access.AddTags(digest, tags...)
}

func (n *Namespace) AddBlob(blob cpi.BlobAccess) error {
	return n.access.AddBlob(blob)
}
