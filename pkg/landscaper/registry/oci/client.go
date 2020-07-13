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
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/containerd/containerd/remotes"
	"github.com/deislabs/oras/pkg/auth"
	orascontent "github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/json"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/utils/oci"
)

// Client is a OCI registry client to handle interaction with a OCI compliant registry.
type Client interface {
	Pull(ctx context.Context, ref string) (*lsv1alpha1.ComponentDefinition, error)
	Push(ctx context.Context, def *lsv1alpha1.ComponentDefinition) error
}

type client struct {
	authorizer auth.Client
	resolver   remotes.Resolver
}

// NewClient creates a new OCI registry client
func NewClient(authorizer auth.Client) (Client, error) {
	resolver, err := authorizer.Resolver(context.Background(), http.DefaultClient, false)
	if err != nil {
		return nil, err
	}
	return &client{
		authorizer: authorizer,
		resolver:   resolver,
	}, nil
}

// Pull loads a ComponentDefinition from a registry
func (c *client) Pull(ctx context.Context, ref string) (*lsv1alpha1.ComponentDefinition, error) {
	ingester := orascontent.NewMemoryStore()
	desc, _, err := oras.Pull(ctx, c.resolver, ref, ingester,
		oras.WithPullEmptyNameAllowed(),
		oras.WithAllowedMediaTypes(KnownMediaTypes()),
		oras.WithContentProvideIngester(ingester))
	if err != nil {
		return nil, err
	}

	manifest, err := oci.ParseManifest(ingester, desc)
	if err != nil {
		return nil, err
	}

	if manifest.Config.MediaType == ComponentDefinitionConfigMediaType {
		return nil, errors.New("unexpected config media type")
	}

	_, blob, ok := ingester.Get(manifest.Config)
	if !ok {
		return nil, errors.New("config not found")
	}

	decoder := serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDeserializer()
	def := &lsv1alpha1.ComponentDefinition{}
	if _, _, err := decoder.Decode(blob, nil, def); err != nil {
		return nil, err
	}
	return def, nil
}

// Push uploads a component definition with its content to a oci compliant registry.
// Push should be used withint the commandline
func (c *client) Push(ctx context.Context, def *lsv1alpha1.ComponentDefinition) error {
	ingester := orascontent.NewMemoryStore()

	data, err := json.Marshal(def)
	if err != nil {
		return err
	}
	desc := ingester.Add("config", ComponentDefinitionConfigMediaType, data)

	_, err = oras.Push(ctx, c.resolver, fmt.Sprintf("%s:%s", def.Name, def.Version), ingester, []ocispecv1.Descriptor{desc})
	return err
}
