// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"bytes"
	"io/ioutil"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

// GetLayerByName returns the layer with a given name.
// The name should be specified by the annotation title.
func GetLayerByName(layers []ocispecv1.Descriptor, name string) *ocispecv1.Descriptor {
	for _, desc := range layers {
		if title, ok := desc.Annotations[ocispecv1.AnnotationTitle]; ok {
			if title == name {
				return &desc
			}
		}
	}
	return nil
}

// BuildTarGzipLayer tar and gzips the given path and adds the layer to the cache.
// It returns the newly creates ocispec Description for the tar.
func BuildTarGzipLayer(cache cache.Cache, fs vfs.FileSystem, path string, annotations map[string]string) (ocispecv1.Descriptor, error) {

	var blob bytes.Buffer
	if err := utils.BuildTarGzip(fs, path, &blob); err != nil {
		return ocispecv1.Descriptor{}, err
	}

	desc := ocispecv1.Descriptor{
		MediaType:   MediaTypeTarGzip,
		Digest:      digest.FromBytes(blob.Bytes()),
		Size:        int64(blob.Len()),
		Annotations: annotations,
	}

	if err := cache.Add(desc, ioutil.NopCloser(&blob)); err != nil {
		return ocispecv1.Descriptor{}, err
	}

	return desc, nil
}
