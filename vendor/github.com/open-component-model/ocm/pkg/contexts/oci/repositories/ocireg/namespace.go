// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocireg

import (
	"context"
	"fmt"

	"github.com/containerd/containerd/errdefs"
	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/oci/actions/oci-repository-prepare"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi/support"
	"github.com/open-component-model/ocm/pkg/docker/resolve"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/logging"
)

type NamespaceContainer struct {
	impl     support.NamespaceAccessImpl
	repo     *RepositoryImpl
	resolver resolve.Resolver
	lister   resolve.Lister
	fetcher  resolve.Fetcher
	pusher   resolve.Pusher
	blobs    *BlobContainers
	checked  bool
}

var _ support.NamespaceContainer = (*NamespaceContainer)(nil)

func NewNamespace(repo *RepositoryImpl, name string) (cpi.NamespaceAccess, error) {
	ref := repo.getRef(name, "")
	resolver, err := repo.getResolver(name)
	if err != nil {
		return nil, err
	}
	fetcher, err := resolver.Fetcher(context.Background(), ref)
	if err != nil {
		return nil, err
	}
	pusher, err := resolver.Pusher(context.Background(), ref)
	if err != nil {
		return nil, err
	}
	lister, err := resolver.Lister(context.Background(), ref)
	if err != nil {
		return nil, err
	}
	c := &NamespaceContainer{
		repo:     repo,
		resolver: resolver,
		lister:   lister,
		fetcher:  fetcher,
		pusher:   pusher,
		blobs:    NewBlobContainers(repo.GetContext(), fetcher, pusher),
	}
	return support.NewNamespaceAccess(name, c, repo)
}

func (n *NamespaceContainer) Close() error {
	return n.blobs.Release()
}

func (n *NamespaceContainer) SetImplementation(impl support.NamespaceAccessImpl) {
	n.impl = impl
}

func (n *NamespaceContainer) getPusher(vers string) (resolve.Pusher, error) {
	err := n.assureCreated()
	if err != nil {
		return nil, err
	}

	ref := n.repo.getRef(n.impl.GetNamespace(), vers)
	resolver := n.resolver

	n.repo.GetContext().Logger().Trace("get pusher", "ref", ref)
	if ok, _ := artdesc.IsDigest(vers); !ok {
		var err error

		resolver, err = n.repo.getResolver(n.impl.GetNamespace())

		if err != nil {
			return nil, fmt.Errorf("unable get resolver: %w", err)
		}
	}

	return resolver.Pusher(dummyContext, ref)
}

func (n *NamespaceContainer) push(vers string, blob cpi.BlobAccess) error {
	p, err := n.getPusher(vers)
	if err != nil {
		return fmt.Errorf("unable to get pusher: %w", err)
	}
	n.repo.GetContext().Logger().Trace("pushing", "version", vers)
	return push(dummyContext, p, blob)
}

func (n *NamespaceContainer) IsReadOnly() bool {
	return n.repo.IsReadOnly()
}

func (n *NamespaceContainer) GetBlobDescriptor(digest digest.Digest) *cpi.Descriptor {
	return nil
}

func (n *NamespaceContainer) GetBlobData(digest digest.Digest) (int64, cpi.DataAccess, error) {
	n.repo.GetContext().Logger().Debug("getting blob", "digest", digest)
	blob, err := n.blobs.Get("")
	if err != nil {
		return -1, nil, fmt.Errorf("failed to retrieve blob data: %w", err)
	}
	size, acc, err := blob.GetBlobData(digest)
	n.repo.GetContext().Logger().Debug("getting blob done", "digest", digest, "size", size, "error", logging.ErrorMessage(err))
	return size, acc, err
}

func (n *NamespaceContainer) AddBlob(blob cpi.BlobAccess) error {
	log := n.repo.GetContext().Logger()
	log.Debug("adding blob", "digest", blob.Digest())
	blobData, err := n.blobs.Get("")
	if err != nil {
		return fmt.Errorf("failed to retrieve blob data: %w", err)
	}
	err = n.assureCreated()
	if err != nil {
		return err
	}
	if _, _, err := blobData.AddBlob(blob); err != nil {
		log.Debug("adding blob failed", "digest", blob.Digest(), "error", err.Error())
		return fmt.Errorf("unable to add blob (OCI repository %s): %w", n.impl.GetNamespace(), err)
	}
	log.Debug("adding blob done", "digest", blob.Digest())
	return nil
}

func (n *NamespaceContainer) ListTags() ([]string, error) {
	return n.lister.List(dummyContext)
}

func (n *NamespaceContainer) GetArtifact(i support.NamespaceAccessImpl, vers string) (cpi.ArtifactAccess, error) {
	ref := n.repo.getRef(n.impl.GetNamespace(), vers)
	n.repo.GetContext().Logger().Debug("get artifact", "ref", ref)
	_, desc, err := n.resolver.Resolve(context.Background(), ref)
	n.repo.GetContext().Logger().Debug("done", "digest", desc.Digest, "size", desc.Size, "mimetype", desc.MediaType, "error", logging.ErrorMessage(err))
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil, errors.ErrNotFound(cpi.KIND_OCIARTIFACT, ref, n.impl.GetNamespace())
		}
		return nil, err
	}
	blobData, err := n.blobs.Get(desc.MediaType)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve blob data, blob data was empty: %w", err)
	}
	_, acc, err := blobData.GetBlobData(desc.Digest)
	if err != nil {
		return nil, err
	}
	return support.NewArtifactForBlob(i, accessio.BlobAccessForDataAccess(desc.Digest, desc.Size, desc.MediaType, acc))
}

func (n *NamespaceContainer) HasArtifact(vers string) (bool, error) {
	ref := n.repo.getRef(n.impl.GetNamespace(), vers)
	n.repo.GetContext().Logger().Debug("check artifact", "ref", ref)
	_, desc, err := n.resolver.Resolve(context.Background(), ref)
	n.repo.GetContext().Logger().Debug("done", "digest", desc.Digest, "size", desc.Size, "mimetype", desc.MediaType, "error", logging.ErrorMessage(err))
	if err != nil {
		if errdefs.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (n *NamespaceContainer) assureCreated() error {
	if n.checked {
		return nil
	}
	var props common.Properties
	if creds, err := n.repo.getCreds(n.impl.GetNamespace()); err == nil && creds != nil {
		props = creds.Properties()
	}
	r, err := oci_repository_prepare.Execute(n.repo.GetContext().GetActions(), n.repo.info.HostPort(), n.impl.GetNamespace(), props)
	n.checked = true
	if err != nil {
		return err
	}
	if r != nil {
		n.repo.GetContext().Logger().Debug("prepare action executed", "message", r.Message)
	}
	return nil
}

func (n *NamespaceContainer) AddArtifact(artifact cpi.Artifact, tags ...string) (access accessio.BlobAccess, err error) {
	blob, err := artifact.Blob()
	if err != nil {
		return nil, err
	}

	if n.repo.info.Legacy {
		blob = artdesc.MapArtifactBlobMimeType(blob, true)
	}

	n.repo.GetContext().Logger().Debug("adding artifact", "digest", blob.Digest(), "mimetype", blob.MimeType())
	blobData, err := n.blobs.Get(blob.MimeType())
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve blob data: %w", err)
	}

	_, _, err = blobData.AddBlob(blob)
	if err != nil {
		return nil, err
	}

	if len(tags) > 0 {
		for _, tag := range tags {
			if err := n.push(tag, blob); err != nil {
				return nil, err
			}
		}
	}

	return blob, err
}

func (n *NamespaceContainer) AddTags(digest digest.Digest, tags ...string) error {
	_, desc, err := n.resolver.Resolve(context.Background(), n.repo.getRef(n.impl.GetNamespace(), digest.String()))
	if err != nil {
		return fmt.Errorf("unable to resolve: %w", err)
	}

	acc, err := NewDataAccess(n.fetcher, desc.Digest, desc.MediaType, false)
	if err != nil {
		return fmt.Errorf("error creating new data access: %w", err)
	}

	blob := accessio.BlobAccessForDataAccess(desc.Digest, desc.Size, desc.MediaType, acc)
	for _, tag := range tags {
		err := n.push(tag, blob)
		if err != nil {
			return fmt.Errorf("unable to push: %w", err)
		}
	}

	return nil
}

func (n *NamespaceContainer) NewArtifact(i support.NamespaceAccessImpl, art ...*artdesc.Artifact) (cpi.ArtifactAccess, error) {
	if n.IsReadOnly() {
		return nil, accessio.ErrReadOnly
	}
	return support.NewArtifact(i, art...)
}
