// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
)

type BlobSink interface {
	AddBlob(blob accessio.BlobAccess) error
}

// StorageContext is the context information passed for Blobhandler
// registered for context type oci.CONTEXT_TYPE.
type StorageContext interface {
	cpi.StorageContext
	BlobSink
}

type DefaultStorageContext struct {
	cpi.DefaultStorageContext
	Sink BlobSink
}

func New(repo cpi.Repository, vers cpi.ComponentVersionAccess, access BlobSink, impltyp string) StorageContext {
	return &DefaultStorageContext{
		DefaultStorageContext: *cpi.NewDefaultStorageContext(repo, vers, cpi.ImplementationRepositoryType{cpi.CONTEXT_TYPE, impltyp}),
		Sink:                  access,
	}
}

func (c *DefaultStorageContext) AddBlob(blob accessio.BlobAccess) error {
	return c.Sink.AddBlob(blob)
}
