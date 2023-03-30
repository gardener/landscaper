// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"reflect"

	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	ocmcpi "github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/genericocireg/componentmapping"
)

// StorageContext is the context information passed for Blobhandler
// registered for context type oci.CONTEXT_TYPE.
type StorageContext struct {
	ocmcpi.DefaultStorageContext
	Repository cpi.Repository
	Namespace  cpi.NamespaceAccess
	Manifest   cpi.ManifestAccess
}

var _ ocmcpi.StorageContext = (*StorageContext)(nil)

func New(comprepo ocmcpi.Repository, vers ocmcpi.ComponentVersionAccess, impltyp string, ocirepo oci.Repository, namespace oci.NamespaceAccess, manifest oci.ManifestAccess) *StorageContext {
	return &StorageContext{
		DefaultStorageContext: *ocmcpi.NewDefaultStorageContext(
			comprepo,
			vers,
			ocmcpi.ImplementationRepositoryType{
				ContextType:    cpi.CONTEXT_TYPE,
				RepositoryType: impltyp,
			},
		),
		Repository: ocirepo,
		Namespace:  namespace,
		Manifest:   manifest,
	}
}

func (s *StorageContext) TargetComponentRepository() ocmcpi.Repository {
	return s.ComponentRepository
}

func (s *StorageContext) TargetComponentVersion() ocmcpi.ComponentVersionAccess {
	return s.ComponentVersion
}

func (s *StorageContext) AssureLayer(blob cpi.BlobAccess) error {
	d := artdesc.DefaultBlobDescriptor(blob)
	desc := s.Manifest.GetDescriptor()

	found := -1
	for i, l := range desc.Layers {
		if reflect.DeepEqual(&desc.Layers[i], d) {
			return nil
		}
		if l.Digest == blob.Digest() {
			found = i
		}
	}
	if found > 0 { // ignore layer 0 used for component descriptor
		desc.Layers[found] = *d
	} else {
		if len(desc.Layers) == 0 {
			// fake descriptor layer
			desc.Layers = append(desc.Layers, ociv1.Descriptor{MediaType: componentmapping.ComponentDescriptorConfigMimeType})
		}
		desc.Layers = append(desc.Layers, *d)
	}
	return nil
}
