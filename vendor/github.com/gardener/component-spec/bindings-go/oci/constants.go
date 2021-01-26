// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package oci

// ComponentDescriptorTarMimeType is the old mimetype for component-descriptor-blobs
// that are stored as tar.
const ComponentDescriptorTarMimeType = "application/vnd.gardener.cloud.cnudie.component-descriptor.v2+yaml+tar"

// LegacyComponentDescriptorTarMimeType is the legacy mimetype for component-descriptor-blobs
// that are stored as tar.
const LegacyComponentDescriptorTarMimeType = "application/vnd.oci.gardener.cloud.cnudie.component-descriptor.config.v2+yaml+tar"

// ComponentDescriptorJSONMimeType is the mimetype for component-descriptor-blobs
// that are stored as JSON.
const ComponentDescriptorJSONMimeType = "application/vnd.gardener.cloud.cnudie.component-descriptor.v2+json"

// ComponentDescriptorNamespace is the subpath for all component descriptor artifacts in an oci registry.‚
const ComponentDescriptorNamespace = "component-descriptors"

// ComponentDescriptorMimeType are the mimetypes for component-descriptor-blobs.
var ComponentDescriptorMimeType = []string{
	ComponentDescriptorTarMimeType,
	ComponentDescriptorJSONMimeType,
}

// ComponentDescriptorConfigMimeType is the mimetype for component-descriptor-oci-cfg-blobs.
const ComponentDescriptorConfigMimeType = "application/vnd.gardener.cloud.cnudie.component.config.v1+json"

// ComponentDescriptorLegacyConfigMimeType is the mimetype for the legacy component-descriptor-oci-cfg-blobs
const ComponentDescriptorLegacyConfigMimeType = "application/vnd.oci.gardener.cloud.cnudie.component-descriptor-metadata.config.v2+json"
