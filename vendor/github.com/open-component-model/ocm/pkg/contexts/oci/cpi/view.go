// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"fmt"
	"io"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessio/resource"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/internal"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
)

var ErrClosed = resource.ErrClosed

////////////////////////////////////////////////////////////////////////////////

type _RepositoryView interface {
	resource.ResourceViewInt[Repository] // here you have to redeclare
}

type RepositoryViewManager = resource.ViewManager[Repository] // here you have to use an alias

type RepositoryImpl interface {
	internal.RepositoryImpl
	resource.ResourceImplementation[Repository]
}

type _RepositoryImplBase = resource.ResourceImplBase[Repository]

type RepositoryImplBase struct {
	_RepositoryImplBase
	ctx Context
}

func (b *RepositoryImplBase) GetContext() Context {
	return b.ctx
}

func NewRepositoryImplBase(ctx Context) RepositoryImplBase {
	return RepositoryImplBase{
		_RepositoryImplBase: resource.ResourceImplBase[Repository]{},
		ctx:                 ctx,
	}
}

type repositoryView struct {
	_RepositoryView
	impl RepositoryImpl
}

var (
	_ Repository                        = (*repositoryView)(nil)
	_ internal.ConsumerIdentityProvider = (*repositoryView)(nil)
)

func GetRepositoryImplementation(n Repository) (RepositoryImpl, error) {
	if v, ok := n.(*repositoryView); ok {
		return v.impl, nil
	}
	return nil, errors.ErrNotSupported("repository implementation type", fmt.Sprintf("%T", n))
}

func repositoryViewCreator(i RepositoryImpl, v resource.CloserView, d RepositoryViewManager) Repository {
	return &repositoryView{
		_RepositoryView: resource.NewView[Repository](v, d),
		impl:            i,
	}
}

func NewRepository(impl RepositoryImpl, name ...string) Repository {
	return resource.NewResource[Repository](impl, repositoryViewCreator, utils.OptionalDefaulted("OCI repo", name...), true)
}

func (r *repositoryView) GetConsumerId(uctx ...credentials.UsageContext) credentials.ConsumerIdentity {
	return credentials.GetProvidedConsumerId(r.impl, uctx...)
}

func (r *repositoryView) GetIdentityMatcher() string {
	return credentials.GetProvidedIdentityMatcher(r.impl)
}

func (r *repositoryView) GetSpecification() internal.RepositorySpec {
	return r.impl.GetSpecification()
}

func (r *repositoryView) GetContext() Context {
	return r.impl.GetContext()
}

func (r *repositoryView) NamespaceLister() (lister internal.NamespaceLister) {
	return r.impl.NamespaceLister()
}

func (r *repositoryView) ExistsArtifact(name string, ref string) (ok bool, err error) {
	err = r.Execute(func() error {
		ok, err = r.impl.ExistsArtifact(name, ref)
		return err
	})
	return ok, err
}

func (r *repositoryView) LookupArtifact(name string, ref string) (acc ArtifactAccess, err error) {
	err = r.Execute(func() error {
		acc, err = r.impl.LookupArtifact(name, ref)
		return err
	})
	return acc, err
}

func (r *repositoryView) LookupNamespace(name string) (acc NamespaceAccess, err error) {
	err = r.Execute(func() error {
		acc, err = r.impl.LookupNamespace(name)
		return err
	})
	return acc, err
}

////////////////////////////////////////////////////////////////////////////////

type _NamespaceAccessView interface {
	resource.ResourceViewInt[NamespaceAccess] // here you have to redeclare
}

type NamespaceAccessViewManager = resource.ViewManager[NamespaceAccess] // here you have to use an alias

type NamespaceAccessImpl interface {
	internal.NamespaceAccessImpl

	resource.ResourceImplementation[NamespaceAccess]

	GetNamespace() string
}

type _NamespaceAccessImplBase = resource.ResourceImplBase[NamespaceAccess]

type NamespaceAccessImplBase struct {
	*_NamespaceAccessImplBase
	namespace string
}

func NewNamespaceAccessImplBase(namespace string, repo RepositoryViewManager, closer ...io.Closer) (*NamespaceAccessImplBase, error) {
	base, err := resource.NewResourceImplBase[NamespaceAccess](repo, closer...)
	if err != nil {
		return nil, err
	}
	return &NamespaceAccessImplBase{
		_NamespaceAccessImplBase: base,
		namespace:                namespace,
	}, nil
}

func (b *NamespaceAccessImplBase) GetNamespace() string {
	return b.namespace
}

type namespaceAccessView struct {
	_NamespaceAccessView
	impl NamespaceAccessImpl
}

var _ NamespaceAccess = (*namespaceAccessView)(nil)

func GetNamespaceAccessImplementation(n NamespaceAccess) (NamespaceAccessImpl, error) {
	if v, ok := n.(*namespaceAccessView); ok {
		return v.impl, nil
	}
	return nil, errors.ErrNotSupported("namespace implementation type", fmt.Sprintf("%T", n))
}

func namespaceAccessViewCreator(i NamespaceAccessImpl, v resource.CloserView, d NamespaceAccessViewManager) NamespaceAccess {
	return &namespaceAccessView{
		_NamespaceAccessView: resource.NewView[NamespaceAccess](v, d),
		impl:                 i,
	}
}

func NewNamespaceAccess(impl NamespaceAccessImpl, kind ...string) NamespaceAccess {
	return resource.NewResource[NamespaceAccess](impl, namespaceAccessViewCreator, fmt.Sprintf("%s %s", utils.OptionalDefaulted("namespace", kind...), impl.GetNamespace()), true)
}

func (n *namespaceAccessView) GetNamespace() string {
	return n.impl.GetNamespace()
}

func (n *namespaceAccessView) GetArtifact(version string) (acc internal.ArtifactAccess, err error) {
	err = n.Execute(func() error {
		acc, err = n.impl.GetArtifact(version)
		return err
	})
	return acc, err
}

func (n *namespaceAccessView) GetBlobData(digest digest.Digest) (size int64, acc internal.DataAccess, err error) {
	err = n.Execute(func() error {
		size, acc, err = n.impl.GetBlobData(digest)
		return err
	})
	return size, acc, err
}

func (n *namespaceAccessView) AddBlob(access internal.BlobAccess) error {
	return n.Execute(func() error {
		return n.impl.AddBlob(access)
	})
}

func (n *namespaceAccessView) HasArtifact(vers string) (ok bool, err error) {
	err = n.Execute(func() error {
		ok, err = n.impl.HasArtifact(vers)
		return err
	})
	return ok, err
}

func (n *namespaceAccessView) AddArtifact(a internal.Artifact, tags ...string) (acc internal.BlobAccess, err error) {
	err = n.Execute(func() error {
		acc, err = n.impl.AddArtifact(a, tags...)
		return err
	})
	return acc, err
}

func (n *namespaceAccessView) AddTags(digest digest.Digest, tags ...string) error {
	return n.Execute(func() error {
		return n.impl.AddTags(digest, tags...)
	})
}

func (n *namespaceAccessView) ListTags() (list []string, err error) {
	err = n.Execute(func() error {
		list, err = n.impl.ListTags()
		return err
	})
	return list, err
}

func (n *namespaceAccessView) NewArtifact(artifact ...*artdesc.Artifact) (acc internal.ArtifactAccess, err error) {
	err = n.Execute(func() error {
		acc, err = n.impl.NewArtifact(artifact...)
		return err
	})
	return acc, err
}

////////////////////////////////////////////////////////////////////////////////

type _ArtifactAccessView interface {
	resource.ResourceViewInt[ArtifactAccess]
}

type ArtifactAccessViewManager = resource.ViewManager[ArtifactAccess]

type ArtifactAccessImpl interface {
	internal.ArtifactAccessImpl

	resource.ResourceImplementation[ArtifactAccess]

	// creation of slave objects require the original view they are created for.

	ManifestAccess(ArtifactAccess) ManifestAccess
	IndexAccess(ArtifactAccess) IndexAccess
}

type ArtifactAccessImplBase = resource.ResourceImplBase[ArtifactAccess]

func NewArtifactAccessImplBase(ns NamespaceAccessViewManager, closer ...io.Closer) (*ArtifactAccessImplBase, error) {
	return resource.NewResourceImplBase[ArtifactAccess](ns, closer...)
}

type artifactAccessView struct {
	_ArtifactAccessView
	impl ArtifactAccessImpl
}

var _ ArtifactAccess = (*artifactAccessView)(nil)

func artifactAccessViewCreator(i ArtifactAccessImpl, v resource.CloserView, d resource.ViewManager[ArtifactAccess]) ArtifactAccess {
	return &artifactAccessView{
		_ArtifactAccessView: resource.NewView[ArtifactAccess](v, d),
		impl:                i,
	}
}

func NewArtifactAccess(impl ArtifactAccessImpl) ArtifactAccess {
	return resource.NewResource[ArtifactAccess](impl, artifactAccessViewCreator, "artifact", true)
}

func (a *artifactAccessView) IsManifest() bool {
	return a.impl.IsManifest()
}

func (a *artifactAccessView) IsIndex() bool {
	return a.impl.IsIndex()
}

func (a *artifactAccessView) Digest() digest.Digest {
	return a.impl.Digest()
}

func (a *artifactAccessView) Blob() (internal.BlobAccess, error) {
	return a.impl.Blob()
}

func (a *artifactAccessView) GetDescriptor() *artdesc.Artifact {
	return a.impl.GetDescriptor()
}

func (a *artifactAccessView) Artifact() *artdesc.Artifact {
	return a.impl.Artifact()
}

func (a *artifactAccessView) Manifest() (*artdesc.Manifest, error) {
	return a.impl.Manifest()
}

func (a *artifactAccessView) ManifestAccess() internal.ManifestAccess {
	return a.impl.ManifestAccess(a)
}

func (a *artifactAccessView) Index() (*artdesc.Index, error) {
	return a.impl.Index()
}

func (a *artifactAccessView) IndexAccess() internal.IndexAccess {
	return a.impl.IndexAccess(a)
}

func (a *artifactAccessView) GetBlobData(digest digest.Digest) (size int64, acc internal.DataAccess, err error) {
	size = -1
	err = a.Execute(func() error {
		size, acc, err = a.impl.GetBlobData(digest)
		return err
	})
	return size, acc, err
}

func (a *artifactAccessView) AddBlob(access internal.BlobAccess) error {
	return a.Execute(func() error {
		return a.impl.AddBlob(access)
	})
}

func (a *artifactAccessView) GetBlob(digest digest.Digest) (acc internal.BlobAccess, err error) {
	err = a.Execute(func() error {
		acc, err = a.impl.GetBlob(digest)
		return err
	})
	return acc, err
}

func (a *artifactAccessView) GetArtifact(digest digest.Digest) (acc internal.ArtifactAccess, err error) {
	err = a.Execute(func() error {
		acc, err = a.impl.GetArtifact(digest)
		return err
	})
	return acc, err
}

func (a *artifactAccessView) AddArtifact(artifact internal.Artifact, platform *artdesc.Platform) (acc internal.BlobAccess, err error) {
	err = a.Execute(func() error {
		acc, err = a.impl.AddArtifact(artifact, platform)
		return err
	})
	return acc, err
}

func (a *artifactAccessView) NewArtifact(art ...*artdesc.Artifact) (acc ArtifactAccess, err error) {
	err = a.Execute(func() error {
		acc, err = a.impl.NewArtifact(art...)
		return err
	})
	return acc, err
}

func (a *artifactAccessView) AddLayer(access internal.BlobAccess, descriptor *artdesc.Descriptor) (index int, err error) {
	index = -1
	err = a.Execute(func() error {
		index, err = a.impl.AddLayer(access, descriptor)
		return err
	})
	return index, err
}
