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
	"bytes"
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/utils/oci"
	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

// BuildNewDefinition creates a ocispec Manifest from a component definition.
func BuildNewDefinition(cache cache.Cache, fs afero.Fs, path string) (*ocispecv1.Manifest, error) {

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
func BuildNewDefinitionConfig(cache cache.Cache, fs afero.Fs, path string) (ocispecv1.Descriptor, error) {
	data, err := afero.ReadFile(fs, filepath.Join(path, lsv1alpha1.ComponentDefinitionPath))
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
		MediaType: ComponentDefinitionConfigMediaType,
		Digest:    digest.FromBytes(data),
		Size:      int64(len(data)),
	}

	if err := cache.Add(desc, ioutil.NopCloser(bytes.NewBuffer(data))); err != nil {
		return ocispecv1.Descriptor{}, err
	}
	return desc, nil
}

// BuildNewContentBlob creates a ocispec Manifest from a component definition.
func BuildNewContentBlob(cache cache.Cache, fs afero.Fs, path string) (ocispecv1.Descriptor, error) {
	ann := map[string]string{
		ocispecv1.AnnotationTitle: ComponentDefinitionAnnotationTitleContent,
	}
	return oci.BuildTarGzipLayer(cache, fs, path, ann)
}
