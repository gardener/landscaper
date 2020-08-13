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
	"context"
	"errors"
	"fmt"
	"io/ioutil"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/gardener/landscaper/pkg/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	regapi "github.com/gardener/landscaper/pkg/landscaper/registry"
	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/oci"
)

type registry struct {
	oci oci.Client
	dec runtime.Decoder
}

// New creates a new oci registry from a oci config.
func New(log logr.Logger, config *config.OCIConfiguration) (regapi.Registry, error) {
	client, err := oci.NewClient(log, oci.WithConfiguration(config))
	if err != nil {
		return nil, err
	}

	return &registry{
		oci: client,
		dec: serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDecoder(),
	}, nil
}

// NewWithOCIClient creates a new oci registry with a oci client
func NewWithOCIClient(log logr.Logger, client oci.Client) (regapi.Registry, error) {
	return &registry{
		oci: client,
		dec: serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDecoder(),
	}, nil
}

// GetDefinition returns the definition for a specific name, version and type.
func (r *registry) GetDefinition(ctx context.Context, ref cdv2.Resource) (*lsv1alpha1.Blueprint, error) {
	if ref.Access.GetType() != cdv2.OCIRegistryType {
		return nil, regapi.NewWrongTypeError(ref.Access.GetType(), ref.Name, ref.Version, nil)
	}
	ociComp := ref.Access.(*cdv2.OCIRegistryAccess)
	ociRef := ociComp.ImageReference

	manifest, err := r.oci.GetManifest(ctx, ociRef)
	if err != nil {
		return nil, err
	}

	if manifest.Config.MediaType != ComponentDefinitionConfigMediaType {
		return nil, fmt.Errorf("expected media type %s but got %s", ComponentDefinitionConfigMediaType, manifest.Config.MediaType)
	}

	// manifest config should contain the component definition
	var configdata bytes.Buffer
	if err := r.oci.Fetch(ctx, ociRef, manifest.Config, &configdata); err != nil {
		return nil, err
	}

	def := &lsv1alpha1.Blueprint{}
	if _, _, err := r.dec.Decode(configdata.Bytes(), nil, def); err != nil {
		return nil, err
	}

	return def, nil
}

// GetBlob returns the blob content for a component definition.
func (r *registry) GetContent(ctx context.Context, ref cdv2.Resource) (afero.Fs, error) {
	if ref.Access.GetType() != cdv2.OCIRegistryType {
		return nil, regapi.NewWrongTypeError(ref.Access.GetType(), ref.Name, ref.Version, nil)
	}
	ociComp := ref.Access.(*cdv2.OCIRegistryAccess)
	ociRef := ociComp.ImageReference

	manifest, err := r.oci.GetManifest(ctx, ociRef)
	if err != nil {
		return nil, err
	}

	layer := oci.GetLayerByName(manifest.Layers, ComponentDefinitionAnnotationTitleContent)
	if layer == nil {
		return nil, regapi.NewNotFoundError(ociRef, errors.New("no content defined for component"))
	}

	var blob bytes.Buffer
	if err := r.oci.Fetch(ctx, ociRef, *layer, &blob); err != nil {
		return nil, err
	}

	// todo: use proper cache folder
	tmpDir, err := ioutil.TempDir("", "content-")
	if err != nil {
		return nil, err
	}
	fs := afero.NewReadOnlyFs(afero.NewBasePathFs(afero.NewOsFs(), tmpDir))

	if err := utils.ExtractTarGzip(&blob, fs, "/"); err != nil {
		return nil, err
	}

	return fs, nil
}
