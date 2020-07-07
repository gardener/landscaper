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
	"errors"

	"github.com/containerd/containerd/content"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

type Store interface {
	content.Ingester
	content.Provider
	Get(desc ocispecv1.Descriptor) (ocispecv1.Descriptor, []byte, bool)
}

// ParseManifest parses a oci manifest from a description and a store.
func ParseManifest(store Store, desc ocispecv1.Descriptor) (*ocispecv1.Manifest, error) {
	_, data, ok := store.Get(desc)
	if !ok {
		return nil, errors.New("not exist")
	}

	var manifest ocispecv1.Manifest
	err := json.Unmarshal(data, &manifest)
	if err != nil {
		return nil, err
	}
	return &manifest, nil
}
