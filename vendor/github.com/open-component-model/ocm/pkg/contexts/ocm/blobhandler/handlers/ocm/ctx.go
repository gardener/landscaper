// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/utils"
)

type BlobSink interface {
	AddBlob(blob accessio.BlobAccess) (string, error)
}

// StorageContext is the context information passed for Blobhandler
// registered for context type oci.CONTEXT_TYPE.
type StorageContext interface {
	cpi.StorageContext
	BlobSink
}

type DefaultStorageContext struct {
	cpi.DefaultStorageContext
	Sink    BlobSink
	Payload interface{}
}

func New(repo cpi.Repository, vers cpi.ComponentVersionAccess, access BlobSink, impltyp string, payload ...interface{}) StorageContext {
	return &DefaultStorageContext{
		DefaultStorageContext: *cpi.NewDefaultStorageContext(repo, vers, cpi.ImplementationRepositoryType{cpi.CONTEXT_TYPE, impltyp}),
		Sink:                  access,
		Payload:               utils.Optional(payload...),
	}
}

func (c *DefaultStorageContext) AddBlob(blob accessio.BlobAccess) (string, error) {
	return c.Sink.AddBlob(blob)
}
