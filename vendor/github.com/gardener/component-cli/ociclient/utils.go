// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociclient

import (
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
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

// GetLayerByMediaType returns the layers with a given mediatype.
func GetLayerByMediaType(layers []ocispecv1.Descriptor, mediaType string) []ocispecv1.Descriptor {
	descs := make([]ocispecv1.Descriptor, 0)
	for _, desc := range layers {
		if desc.MediaType == mediaType {
			descs = append(descs, desc)
		}
	}
	return descs
}
