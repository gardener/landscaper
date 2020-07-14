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

	"github.com/go-logr/logr"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/gardener/landscaper/pkg/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	regapi "github.com/gardener/landscaper/pkg/landscaper/registry"
	"github.com/gardener/landscaper/pkg/utils/oci"
)

type registry struct {
	oci oci.Client
	dec runtime.Decoder
}

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

func (r registry) GetDefinition(ctx context.Context, name, version string) (*lsv1alpha1.ComponentDefinition, error) {
	return r.GetDefinitionByRef(ctx, ref(name, version))
}

func (r registry) GetDefinitionByRef(ctx context.Context, ref string) (*lsv1alpha1.ComponentDefinition, error) {
	manifest, err := r.oci.GetManifest(ctx, ref)
	if err != nil {
		return nil, err
	}

	if manifest.Config.MediaType != ComponentDefinitionConfigMediaType {
		return nil, fmt.Errorf("expected media type %s but got %s", ComponentDefinitionConfigMediaType, manifest.Config.MediaType)
	}

	// manifest config should contain the component definition
	var configdata bytes.Buffer
	if err := r.oci.Fetch(ctx, ref, manifest.Config, &configdata); err != nil {
		return nil, err
	}

	config := &lsv1alpha1.ComponentDefinition{}
	if _, _, err := r.dec.Decode(configdata.Bytes(), nil, config); err != nil {
		return nil, err
	}

	return nil, nil
}

func (r registry) GetBlob(ctx context.Context, name, version string) (afero.Fs, error) {
	ref := ref(name, version)
	manifest, err := r.oci.GetManifest(ctx, ref)
	if err != nil {
		return nil, err
	}

	if len(manifest.Layers) != 1 {
		return nil, errors.New("unexpected number of layers in manifest")
	}
	layer := manifest.Layers[0]
	if layer.MediaType != ComponentDefinitionContentLayerMediaType {
		return nil, fmt.Errorf("expected media type %s but got %s", ComponentDefinitionContentLayerMediaType, layer.MediaType)
	}

	var blob bytes.Buffer
	if err := r.oci.Fetch(ctx, ref, layer, &blob); err != nil {
		return nil, err
	}
	// todo unzip blob and return created filesystem
	return nil, nil
}

func (r registry) GetVersions(ctx context.Context, name string) ([]string, error) {
	panic("implement me")
}
