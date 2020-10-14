// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bputils

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	blueprintsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/blueprints"
	"github.com/gardener/landscaper/pkg/utils/oci"
	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

// BuildNewDefinition creates a ocispec Manifest from a component definition.
func BuildNewDefinition(cache cache.Cache, fs vfs.FileSystem, path string) (*ocispecv1.Manifest, error) {

	config, err := BuildNewDefinitionConfig(cache, fs, path)
	if err != nil {
		return nil, err
	}

	defLayer, err := BuildNewContentBlob(cache, fs, path)
	if err != nil {
		return nil, err
	}

	manifest := &ocispecv1.Manifest{
		Versioned: specs.Versioned{SchemaVersion: 2},
		Config:    config,
		Layers: []ocispecv1.Descriptor{
			defLayer,
		},
	}

	return manifest, nil
}

// BuildNewDefinitionConfig creates a ocispec Manifest from a component definition.
func BuildNewDefinitionConfig(cache cache.Cache, fs vfs.FileSystem, path string) (ocispecv1.Descriptor, error) {
	data, err := vfs.ReadFile(fs, filepath.Join(path, lsv1alpha1.BlueprintFilePath))
	if err != nil {
		return ocispecv1.Descriptor{}, err
	}

	def := &lsv1alpha1.Blueprint{}
	if _, _, err := serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDecoder().Decode(data, nil, def); err != nil {
		return ocispecv1.Descriptor{}, err
	}

	data, err = json.Marshal(def)
	if err != nil {
		return ocispecv1.Descriptor{}, err
	}

	desc := ocispecv1.Descriptor{
		MediaType: blueprintsregistry.ComponentDefinitionConfigMediaType,
		Digest:    digest.FromBytes(data),
		Size:      int64(len(data)),
	}

	if err := cache.Add(desc, ioutil.NopCloser(bytes.NewBuffer(data))); err != nil {
		return ocispecv1.Descriptor{}, err
	}
	return desc, nil
}

// BuildNewContentBlob creates a ocispec Manifest from a component definition.
func BuildNewContentBlob(cache cache.Cache, fs vfs.FileSystem, path string) (ocispecv1.Descriptor, error) {
	ann := map[string]string{
		ocispecv1.AnnotationTitle: blueprintsregistry.ComponentDefinitionAnnotationTitleContent,
	}
	return oci.BuildTarGzipLayer(cache, fs, path, ann)
}
