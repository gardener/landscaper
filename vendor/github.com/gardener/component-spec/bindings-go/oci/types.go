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

import ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

// OciBlobRef is a single OCI registry layer reference as used in OCI Image Manifests.
type OciBlobRef struct {
	// MediaType is the media type of the object this schema refers to.
	MediaType string `json:"mediaType,omitempty"`

	// Digest is the digest of the targeted content.
	Digest string `json:"digest"`

	// Size specifies the size in bytes of the blob.
	Size int64 `json:"size"`
}

// ConvertDescriptorToOCIBlobRef converts a ocispec descriptor to a component descriptor oci blob ref.
func ConvertDescriptorToOCIBlobRef(desc ocispecv1.Descriptor) OciBlobRef {
	return OciBlobRef{
		MediaType: desc.MediaType,
		Digest:    desc.Digest.String(),
		Size:      desc.Size,
	}
}

// ComponentDescriptorConfig is a Component-Descriptor OCI configuration that is used to store the reference to the
// (pseudo-)layer used to store the Component-Descriptor in.
type ComponentDescriptorConfig struct {
	ComponentDescriptorLayer *OciBlobRef `json:"componentDescriptorLayer,omitempty"`
}
