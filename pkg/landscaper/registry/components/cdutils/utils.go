// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cdutils

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gardener/component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

// BuildComponentDescriptorManifest creates a new manifest from a component descriptor
func BuildComponentDescriptorManifest(cache cache.Cache, cdData []byte) (ocispecv1.Manifest, error) {
	// add dummy json as config to have valid json that is required in a oci manifest
	dummyData := []byte("{}")
	dummyDesc := ocispecv1.Descriptor{
		MediaType: "application/json",
		Digest:    digest.FromBytes(dummyData),
		Size:      int64(len(dummyData)),
	}
	if err := cache.Add(dummyDesc, ioutil.NopCloser(bytes.NewBuffer(dummyData))); err != nil {
		return ocispecv1.Manifest{}, nil
	}

	memFs := memoryfs.New()
	if err := vfs.WriteFile(memFs, filepath.Join("/", ctf.ComponentDescriptorFileName), cdData, os.ModePerm); err != nil {
		return ocispecv1.Manifest{}, err
	}
	var blob bytes.Buffer
	if err := utils.BuildTar(memFs, "/", &blob); err != nil {
		return ocispecv1.Manifest{}, err
	}

	desc := ocispecv1.Descriptor{
		MediaType: cdoci.ComponentDescriptorTarMimeType,
		Digest:    digest.FromBytes(blob.Bytes()),
		Size:      int64(blob.Len()),
	}

	if err := cache.Add(desc, ioutil.NopCloser(&blob)); err != nil {
		return ocispecv1.Manifest{}, err
	}

	manifest := ocispecv1.Manifest{
		Versioned: specs.Versioned{SchemaVersion: 2},
		Config:    dummyDesc,
		Layers: []ocispecv1.Descriptor{
			desc,
		},
	}

	return manifest, nil
}

// BuildNewDefinition creates a ocispec Manifest from a component definition.
func BuildNewManifest(cache cache.Cache, data []byte) (*ocispecv1.Manifest, error) {
	memfs := memoryfs.New()
	if err := vfs.WriteFile(memfs, filepath.Join("/", ctf.ComponentDescriptorFileName), data, os.ModePerm); err != nil {
		return nil, fmt.Errorf("unable to write component descriptor to memory fs: %w", err)
	}

	var blob bytes.Buffer
	if err := utils.BuildTar(memfs, "/", &blob); err != nil {
		return nil, err
	}

	desc := ocispecv1.Descriptor{
		MediaType: cdoci.ComponentDescriptorTarMimeType,
		Digest:    digest.FromBytes(blob.Bytes()),
		Size:      int64(blob.Len()),
	}
	if err := cache.Add(desc, ioutil.NopCloser(&blob)); err != nil {
		return nil, fmt.Errorf("unable to add layer to internal cache: %w", err)
	}

	dummyDesc, err := AddDummyDescriptor(cache)
	if err != nil {
		return nil, fmt.Errorf("unable to add dummy descriptor: %w", err)
	}

	manifest := &ocispecv1.Manifest{
		Versioned: specs.Versioned{SchemaVersion: 2},
		Config:    dummyDesc,
		Layers: []ocispecv1.Descriptor{
			desc,
		},
	}

	return manifest, nil
}

// AddDummyDescriptor adds a empty json dummy descriptor.
func AddDummyDescriptor(c cache.Cache) (ocispecv1.Descriptor, error) {
	dummyData := []byte("{}")
	dummyDesc := ocispecv1.Descriptor{
		MediaType: "application/json",
		Digest:    digest.FromBytes(dummyData),
		Size:      int64(len(dummyData)),
	}
	if err := c.Add(dummyDesc, ioutil.NopCloser(bytes.NewBuffer(dummyData))); err != nil {
		return ocispecv1.Descriptor{}, err
	}
	return dummyDesc, nil
}
