// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package support

import (
	"io"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
)

// BlobContainer is the interface for an element capable to store blobs.
type BlobContainer interface {
	GetBlobData(name string) (cpi.DataAccess, error)

	// GetStorageContext creates a storage context for blobs
	// that is used to feed blob handlers for specific blob storage methods.
	// If no handler accepts the blob, the AddBlobFor method will
	// be used to store the blob
	GetStorageContext(cv cpi.ComponentVersionAccess) cpi.StorageContext

	// AddBlobFor stores a local blob together with the component and
	// potentially provides a global reference according to the OCI distribution spec
	// if the blob described an oci artifact.
	// The resulting access information (global and local) is provided as
	// an access method specification usable in a component descriptor.
	// This is the direct technical storage, without caring about any handler.
	AddBlobFor(storagectx cpi.StorageContext, blob cpi.BlobAccess, refName string, global cpi.AccessSpec) (cpi.AccessSpec, error)
}

// ComponentVersionContainer is the interface of an element hosting a component version.
type ComponentVersionContainer interface {
	SetImplementation(impl ComponentVersionAccessImpl)

	GetParentViewManager() cpi.ComponentAccessViewManager

	GetContext() cpi.Context
	Repository() cpi.Repository

	IsReadOnly() bool
	Update() error

	GetDescriptor() *compdesc.ComponentDescriptor
	BlobContainer
	AccessMethod(a cpi.AccessSpec) (cpi.AccessMethod, error)
	GetInexpensiveContentVersionIdentity(a cpi.AccessSpec) string

	io.Closer
}
