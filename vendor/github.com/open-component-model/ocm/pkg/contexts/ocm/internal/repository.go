// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"io"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessio/resource"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
)

type ComponentVersionResolver interface {
	LookupComponentVersion(name string, version string) (ComponentVersionAccess, error)
}

type RepositoryImpl interface {
	GetContext() Context

	GetSpecification() RepositorySpec
	ComponentLister() ComponentLister

	ExistsComponentVersion(name string, version string) (bool, error)
	LookupComponentVersion(name string, version string) (ComponentVersionAccess, error)
	LookupComponent(name string) (ComponentAccess, error)

	Close() error
}

type Repository interface {
	resource.ResourceView[Repository]

	RepositoryImpl
}

// ConsumerIdentityProvider is an interface for object requiring
// credentials, which want to expose the ConsumerId they are
// usingto request implicit credentials.
type ConsumerIdentityProvider = credentials.ConsumerIdentityProvider

type (
	DataAccess = accessio.DataAccess
	BlobAccess = accessio.BlobAccess
	MimeType   = accessio.MimeType
)

type ComponentAccessImpl interface {
	GetContext() Context
	GetName() string

	ListVersions() ([]string, error)
	LookupVersion(version string) (ComponentVersionAccess, error)
	HasVersion(vers string) (bool, error)
	AddVersion(ComponentVersionAccess) error
	NewVersion(version string, overrides ...bool) (ComponentVersionAccess, error)

	Close() error
}

type ComponentAccess interface {
	resource.ResourceView[ComponentAccess]

	ComponentAccessImpl
}

type (
	ResourceMeta       = compdesc.ResourceMeta
	ComponentReference = compdesc.ComponentReference
)

type BaseAccess interface {
	ComponentVersion() ComponentVersionAccess
	Access() (AccessSpec, error)
	AccessMethod() (AccessMethod, error)
}

type ResourceAccess interface {
	Meta() *ResourceMeta
	BaseAccess
}

type SourceMeta = compdesc.SourceMeta

type SourceAccess interface {
	Meta() *SourceMeta
	BaseAccess
}

type ComponentVersionAccessImpl interface {
	common.VersionedElement

	Repository() Repository

	GetContext() Context

	GetDescriptor() *compdesc.ComponentDescriptor

	SetResource(*ResourceMeta, compdesc.AccessSpec) error
	SetSource(*SourceMeta, compdesc.AccessSpec) error

	SetReference(ref *ComponentReference) error

	DiscardChanges()

	io.Closer
}

type ComponentVersionAccess interface {
	resource.ResourceView[ComponentVersionAccess]

	ComponentVersionAccessImpl

	GetResources() []ResourceAccess
	GetResource(meta metav1.Identity) (ResourceAccess, error)
	GetResourceByIndex(i int) (ResourceAccess, error)
	GetResourcesByName(name string, selectors ...compdesc.IdentitySelector) ([]ResourceAccess, error)
	GetResourcesByIdentitySelectors(selectors ...compdesc.IdentitySelector) ([]ResourceAccess, error)
	GetResourcesByResourceSelectors(selectors ...compdesc.ResourceSelector) ([]ResourceAccess, error)

	GetSources() []SourceAccess
	GetSource(meta metav1.Identity) (SourceAccess, error)
	GetSourceByIndex(i int) (SourceAccess, error)

	GetReference(meta metav1.Identity) (ComponentReference, error)
	GetReferenceByIndex(i int) (ComponentReference, error)
	GetReferencesByName(name string, selectors ...compdesc.IdentitySelector) (compdesc.References, error)
	GetReferencesByIdentitySelectors(selectors ...compdesc.IdentitySelector) (compdesc.References, error)
	GetReferencesByReferenceSelectors(selectors ...compdesc.ReferenceSelector) (compdesc.References, error)

	// AddBlob adds a local blob and returns an appropriate local access spec.
	AddBlob(blob BlobAccess, artType, refName string, global AccessSpec) (AccessSpec, error)

	// AdjustResourceAccess is used to modify the access spec. The old and new one MUST refer to the same content.
	AdjustResourceAccess(meta *ResourceMeta, acc compdesc.AccessSpec) error
	SetResourceBlob(meta *ResourceMeta, blob BlobAccess, refname string, global AccessSpec) error
	AdjustSourceAccess(meta *SourceMeta, acc compdesc.AccessSpec) error
	SetSourceBlob(meta *SourceMeta, blob BlobAccess, refname string, global AccessSpec) error

	// AccessMethod provides an access method implementation for
	// an access spec. This might be a repository local implementation
	// or a global one. It might be called by the AccessSpec method
	// to map itself to a local implementation or called directly.
	// If called it should forward the call to the AccessSpec
	// if and only if this specs NOT states to be local IsLocal()==false
	// If the spec states to be local, the repository is responsible for
	// providing a local implementation or return nil if this is
	// not supported by the actual repository type.
	AccessMethod(AccessSpec) (AccessMethod, error)

	// GetInexpensiveContentVersionIdentity implements a method that attempts to provide an inexpensive identity for
	// the specified artifact. Therefore, an identity that can be provided without requiring the entire object (e.g.
	// calculating the digest from the bytes), which would defeat the purpose of caching.
	// It follows the same contract as AccessMethod.
	GetInexpensiveContentVersionIdentity(spec AccessSpec) string
}

// ComponentLister provides the optional repository list functionality of
// a repository.
type ComponentLister interface {
	// NumComponents returns the number of components found for a prefix
	// If the given prefix does not end with a /, a repository with the
	// prefix name is included
	NumComponents(prefix string) (int, error)

	// GetNamespaces returns the name of namespaces found for a prefix
	// If the given prefix does not end with a /, a repository with the
	// prefix name is included
	GetComponents(prefix string, closure bool) ([]string, error)
}
