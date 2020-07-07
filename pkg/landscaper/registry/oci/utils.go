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
	orascontent "github.com/deislabs/oras/pkg/content"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
)

func getComponentDefinitionFromLayers(ingester *orascontent.Memorystore, layers []ocispecv1.Descriptor) (*lsv1alpha1.ComponentDefinition, error) {
	for _, layer := range layers {
		if layer.MediaType == ComponentDefinitionConfigMediaType {
			_, blob, ok := ingester.Get(layer)
			if !ok {
				return nil, registry.NewComponentNotFoundError(layer.MediaType, nil)
			}

			decoder := serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDeserializer()
			def := &lsv1alpha1.ComponentDefinition{}
			if _, _, err := decoder.Decode(blob, nil, def); err != nil {
				return nil, err
			}
			return def, nil
		}
	}
	return nil, registry.NewComponentNotFoundError("from OCI", nil)
}
