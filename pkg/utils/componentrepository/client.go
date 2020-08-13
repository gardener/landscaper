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

package componentrepository

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/go-logr/logr"

	"github.com/gardener/landscaper/pkg/apis/config"
	"github.com/gardener/landscaper/pkg/utils/oci"
)

var (
	ErrNotFound error = errors.New("NotFound")
)

// ComponentDescriptorMediaType is the media type containing the component descriptor.
const ComponentDescriptorMediaType = "application/sap-cnudie+tar"

// Client is a component descriptor repository implementation
// that resolves component references stored in an oci repository.
type Client struct {
	oci oci.Client
}

// New creates a new oci registry from a oci config.
func New(log logr.Logger, config *config.OCIConfiguration) (*Client, error) {
	client, err := oci.NewClient(log, oci.WithConfiguration(config))
	if err != nil {
		return nil, err
	}

	return &Client{
		oci: client,
	}, nil
}

// NewWithOCIClient creates a new oci registry with a oci client
func NewWithOCIClient(log logr.Logger, client oci.Client) (*Client, error) {
	return &Client{
		oci: client,
	}, nil
}

// Get resolves a reference and returns the component descriptor.
func (r *Client) Get(ctx context.Context, baseURl string, ref cdv2.ObjectMeta) (*cdv2.ComponentDescriptor, error) {
	u, err := url.Parse(baseURl)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, ref.Name)
	refPath := fmt.Sprintf("%s:%s", u.String(), ref.Version)

	manifest, err := r.oci.GetManifest(ctx, refPath)
	if err != nil {
		return nil, err
	}

	if len(manifest.Layers) != 1 {
		return nil, errors.New("manifest must contain 1 layer")
	}
	if manifest.Layers[0].MediaType != ComponentDescriptorMediaType {
		return nil, fmt.Errorf("unexpected media type %s, expected %s", manifest.Layers[0].MediaType, ComponentDescriptorMediaType)
	}

	var data bytes.Buffer
	if err := r.oci.Fetch(ctx, refPath, manifest.Layers[0], &data); err != nil {
		return nil, err
	}

	compDescData, err := readCompDescFromTar(&data)
	if err != nil {
		return nil, err
	}

	cd := &cdv2.ComponentDescriptor{}
	if err := codec.Decode(compDescData, cd); err != nil {
		return nil, err
	}

	return cd, nil
}

func readCompDescFromTar(data io.Reader) ([]byte, error) {
	tr := tar.NewReader(data)
	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				return nil, ErrNotFound
			}
			return nil, err
		}

		if header.Name != "component-descriptor.yaml" {
			continue
		}

		if header.Typeflag == tar.TypeReg {
			var compDescData bytes.Buffer
			if _, err := io.Copy(&compDescData, tr); err != nil {
				return nil, err
			}
			return compDescData.Bytes(), nil
		}
	}
}
