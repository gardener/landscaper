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
	"net/http"

	"github.com/containerd/containerd/remotes"
	auth "github.com/deislabs/oras/pkg/auth/docker"
	"github.com/spf13/afero"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	regapi "github.com/gardener/landscaper/pkg/landscaper/registry"
)

type registry struct {
	resolver remotes.Resolver
	client   Client
}

func New(configFile string) (regapi.Registry, error) {
	authorizer, err := auth.NewClient(configFile)
	if err != nil {
		return nil, err
	}

	resolver, err := authorizer.Resolver(context.Background(), http.DefaultClient, false)
	if err != nil {
		return nil, err
	}

	client, err := NewClient(authorizer)
	if err != nil {
		return nil, err
	}

	return &registry{
		resolver: resolver,
		client:   client,
	}, nil
}

func (r registry) GetDefinition(ctx context.Context, name, version string) (*lsv1alpha1.ComponentDefinition, error) {

	panic("implement me")
}

func (r registry) GetDefinitionByRef(ctx context.Context, ref string) (*lsv1alpha1.ComponentDefinition, error) {
	return r.client.Pull(ctx, ref)
}

func (r registry) GetBlob(ctx context.Context, name, version string) (afero.Fs, error) {
	panic("implement me")
}

func (r registry) GetVersions(ctx context.Context, name string) ([]string, error) {
	panic("implement me")
}
