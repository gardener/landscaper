// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentmapping

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
)

// ComponentDescriptorFileName is the filename of the component descriptor in a tar file used to store
// the component descriptor in an OCI image layer.
const ComponentDescriptorFileName = compdesc.ComponentDescriptorFileName

// ComponentDescriptorTarMimeType is the old mimetype for component-descriptor-blobs
// that are stored as tar.
const ComponentDescriptorTarMimeType = "application/vnd.ocm.software.component-descriptor.v2+yaml+tar"

// Legacy2ComponentDescriptorTarMimeType is the legacy mimetype for component-descriptor-blobs
// that are stored as tar.
const (
	LegacyComponentDescriptorTarMimeType  = "application/vnd.gardener.cloud.cnudie.component-descriptor.v2+yaml+tar"
	Legacy2ComponentDescriptorTarMimeType = "application/vnd.oci.gardener.cloud.cnudie.component-descriptor.config.v2+yaml+tar"
)

// ComponentDescriptorJSONMimeType is the mimetype for component-descriptor-blobs
// that are stored as JSON.
const (
	ComponentDescriptorJSONMimeType       = "application/vnd.ocm.software.component-descriptor.v2+json"
	LegacyComponentDescriptorJSONMimeType = "application/vnd.gardener.cloud.cnudie.component-descriptor.v2+json"
)

// ComponentDescriptorJSONMimeType is the mimetype for component-descriptor-blobs
// that are stored as YAML.
const (
	ComponentDescriptorYAMLMimeType       = "application/vnd.ocm.software.component-descriptor.v2+yaml"
	LegacyComponentDescriptorYAMLMimeType = "application/vnd.gardener.cloud.cnudie.component-descriptor.v2+yaml"
)

// ComponentDescriptorMimeType are the mimetypes for component-descriptor-blobs.
var ComponentDescriptorMimeType = []string{
	ComponentDescriptorTarMimeType,
	Legacy2ComponentDescriptorTarMimeType,
	ComponentDescriptorJSONMimeType,
	LegacyComponentDescriptorJSONMimeType,
}

// ComponentDescriptorConfigMimeType is the mimetype for component-descriptor-oci-cfg-blobs.
const ComponentDescriptorConfigMimeType = "application/vnd.ocm.software.component.config.v1+json"

// LegacyComponentDescriptorConfigMimeType is the mimetype for the legacy component-descriptor-oci-cfg-blobs.
const (
	LegacyComponentDescriptorConfigMimeType  = "application/vnd.gardener.cloud.cnudie.component.config.v1+json"
	Legacy2ComponentDescriptorConfigMimeType = "application/vnd.oci.gardener.cloud.cnudie.component-descriptor-metadata.config.v2+json"
)

// ComponentDescriptorNamespace is the subpath for all component descriptor artifacts in an oci registry.‚.
const ComponentDescriptorNamespace = "component-descriptors"
