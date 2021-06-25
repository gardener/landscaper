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

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/opencontainers/go-digest"
	imagespec "github.com/opencontainers/image-spec/specs-go"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/gardener/component-spec/bindings-go/ctf"
)

// BlobStore defines a interface that is used to store oci descriptors.
type BlobStore interface {
	Add(desc ocispecv1.Descriptor, reader io.ReadCloser) error
}

// ManifestBuilder converts a component descriptor with local defined blobs
// into a oci component descriptor with blobs as layers of the component descriptor.
type ManifestBuilder struct {
	store   BlobStore
	archive *ctf.ComponentArchive
	componentDescriptorStorageType string
}

// NewManifestBuilder creates a new oci manifest builder for a component descriptor
func NewManifestBuilder(store BlobStore, archive *ctf.ComponentArchive) *ManifestBuilder {
	return &ManifestBuilder{
		store:   store,
		archive: archive,
	}
}

func (b *ManifestBuilder) StorageType(storageType string) *ManifestBuilder {
	b.componentDescriptorStorageType = storageType
	return b
}

// Build creates a ocispec Manifest from a component descriptor.
func (b *ManifestBuilder) Build(ctx context.Context) (*ocispecv1.Manifest, error) {
	// default storage type
	if len(b.componentDescriptorStorageType) == 0 {
		b.componentDescriptorStorageType = ComponentDescriptorTarMimeType
	}

	// get additional local artifacts
	additionalBlobDescs, err := b.addLocalBlobs(ctx)
	if err != nil {
		return nil, err
	}

	componentDescriptorDesc, err := b.addComponentDescriptorDesc()
	if err != nil {
		return nil, err
	}

	componentDescriptorLayerOCIRef := ConvertDescriptorToOCIBlobRef(componentDescriptorDesc)
	componentConfig := ComponentDescriptorConfig{
		ComponentDescriptorLayer: &componentDescriptorLayerOCIRef,
	}

	componentConfigBytes, err := json.Marshal(componentConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal component config: %w", err)
	}

	componentConfigDesc := ocispecv1.Descriptor{
		MediaType: ComponentDescriptorConfigMimeType,
		Digest:    digest.FromBytes(componentConfigBytes),
		Size:      int64(len(componentConfigBytes)),
	}
	if err := b.store.Add(componentConfigDesc, ioutil.NopCloser(bytes.NewBuffer(componentConfigBytes))); err != nil {
		return nil, fmt.Errorf("unable to add component config layer to internal store: %w", err)
	}

	manifest := &ocispecv1.Manifest{
		Versioned: imagespec.Versioned{SchemaVersion: 2},
		Config:    componentConfigDesc,
		Layers:    append([]ocispecv1.Descriptor{componentDescriptorDesc}, additionalBlobDescs...),
	}

	return manifest, nil
}

// todo: add support for old tar based components
func (b *ManifestBuilder) addComponentDescriptorDesc() (ocispecv1.Descriptor, error) {
	data, err := codec.Encode(b.archive.ComponentDescriptor)
	if err != nil {
		return ocispecv1.Descriptor{}, fmt.Errorf("unable to encode component descriptor: %w", err)
	}

	if b.componentDescriptorStorageType == ComponentDescriptorJSONMimeType {
		componentDescriptorDesc := ocispecv1.Descriptor{
			MediaType: ComponentDescriptorJSONMimeType,
			Digest:    digest.FromBytes(data),
			Size:      int64(len(data)),
		}
		if err := b.store.Add(componentDescriptorDesc, ioutil.NopCloser(bytes.NewBuffer(data))); err != nil {
			return ocispecv1.Descriptor{}, fmt.Errorf("unable to add component descriptor layer to internal store: %w", err)
		}
		return componentDescriptorDesc, nil
	} else if b.componentDescriptorStorageType == ComponentDescriptorTarMimeType {
		// create tar with component descriptor
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		if err := tw.WriteHeader(&tar.Header{
			Typeflag:   tar.TypeReg,
			Name:       ctf.ComponentDescriptorFileName,
			Size:       int64(len(data)),
			ModTime:    time.Now(),
		}); err != nil {
			return ocispecv1.Descriptor{}, fmt.Errorf("unable to add component descriptor header: %w", err)
		}
		if _, err := io.Copy(tw, bytes.NewBuffer(data)); err != nil {
			return ocispecv1.Descriptor{}, fmt.Errorf("unable to write component-descriptor to tar: %w", err)
		}
		if err := tw.Close(); err != nil {
			return ocispecv1.Descriptor{}, fmt.Errorf("unable to close tar writer: %w", err)
		}

		componentDescriptorDesc := ocispecv1.Descriptor{
			MediaType: ComponentDescriptorTarMimeType,
			Digest:    digest.FromBytes(buf.Bytes()),
			Size:      int64(buf.Len()),
		}
		if err := b.store.Add(componentDescriptorDesc, ioutil.NopCloser(&buf)); err != nil {
			return ocispecv1.Descriptor{}, fmt.Errorf("unable to add component descriptor layer to internal store: %w", err)
		}
		return componentDescriptorDesc, nil
	}

	return ocispecv1.Descriptor{}, fmt.Errorf("unsupported storage type %q", b.componentDescriptorStorageType)
}

// addLocalBlobs adds all local resources to the blob store and updates the component descriptors access method.
func (b *ManifestBuilder) addLocalBlobs(ctx context.Context) ([]ocispecv1.Descriptor, error) {
	blobDescriptors := make([]ocispecv1.Descriptor, 0)

	for i, res := range b.archive.ComponentDescriptor.Resources {
		var blob bytes.Buffer
		info, err := b.archive.Resolve(ctx, res, &blob)
		if err != nil {
			if err == ctf.UnsupportedResolveType {
				continue
			}
			return nil, fmt.Errorf("unable to get blob for resource %s: %w", res.GetName(), err)
		}

		desc := ocispecv1.Descriptor{
			MediaType: info.MediaType,
			Digest:    digest.Digest(info.Digest),
			Size:      info.Size,
		}
		if err := b.store.Add(desc, ioutil.NopCloser(&blob)); err != nil {
			return nil, fmt.Errorf("unable to store blob: %w", err)
		}

		ociBlobAccess := v2.NewLocalOCIBlobAccess(desc.Digest.String())
		unstructuredType, err := v2.NewUnstructured(ociBlobAccess)
		if err != nil {
			return nil, fmt.Errorf("unable to convert ociBlob to untructured type: %w", err)
		}
		res.Access = &unstructuredType
		b.archive.ComponentDescriptor.Resources[i] = res
		blobDescriptors = append(blobDescriptors, desc)
	}
	return blobDescriptors, nil
}
