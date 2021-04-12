// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociclient

import (
	"fmt"
	"strings"

	"github.com/opencontainers/go-digest"
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

// GetLayerByDigest returns the layers with a given digest.
func GetLayerByDigest(layers []ocispecv1.Descriptor, digest string) []ocispecv1.Descriptor {
	descs := make([]ocispecv1.Descriptor, 0)
	for _, desc := range layers {
		if desc.Digest.String() == digest {
			descs = append(descs, desc)
		}
	}
	return descs
}

// ParseImageRef parses a valid image ref into its repository and version
func ParseImageRef(ref string) (repository, version string, err error) {
	// check if the ref contains a digest
	if strings.Contains(ref, "@") {
		splitRef := strings.Split(ref, "@")
		if len(splitRef) != 2 {
			return "", "", fmt.Errorf("invalid image reference %q, expected only 1 char of '@'", ref)
		}
		return splitRef[0], splitRef[1], nil
	}
	splitRef := strings.Split(ref, ":")
	if len(splitRef) > 3 {
		return "", "", fmt.Errorf("invalid image reference %q, expected maximum 3 chars of ':'", ref)
	}

	repository = strings.Join(splitRef[:(len(splitRef)-1)], ":")
	version = splitRef[len(splitRef)-1]
	err = nil
	return
}

// TagIsDigest checks if a tag is a digest.
func TagIsDigest(tag string) bool {
	_, err := digest.Parse(tag)
	return err == nil
}
