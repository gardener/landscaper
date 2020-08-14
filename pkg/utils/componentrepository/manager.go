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
	"context"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

// Client describes a component descriptor repository implementation
// that resolves component references.
type Client interface {
	Resolve(ctx context.Context, repoCtx cdv2.RepositoryContext, ref cdv2.ObjectMeta) (*cdv2.ComponentDescriptor, error)
}

// TypedClient describes a repository ociClient that can handle the given type.
type TypedClient interface {
	Client
	Type() string
}

type repositoryManager map[string]Client

// New creates a ociClient that can handle multiple clients
func New(clients ...TypedClient) Client {
	m := repositoryManager{}

	for _, client := range clients {
		m[client.Type()] = client
	}

	return repositoryManager{}
}

func (m repositoryManager) Resolve(ctx context.Context, repoCtx cdv2.RepositoryContext, ref cdv2.ObjectMeta) (*cdv2.ComponentDescriptor, error) {
	client, ok := m[repoCtx.Type]
	if !ok {
		return nil, fmt.Errorf("unknown repository type %s", repoCtx.Type)
	}
	return client.Resolve(ctx, repoCtx, ref)
}
