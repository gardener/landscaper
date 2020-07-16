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

package operation

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/pkg/landscaper/dataobject"
	"github.com/gardener/landscaper/pkg/landscaper/registry"
)

// Operation is the operation interface that is used to share common operational data across the landscaper reconciler.
type Interface interface {
	Log() logr.Logger
	Client() client.Client
	Scheme() *runtime.Scheme
	Registry() registry.Registry
	GetDataObjectFromSecret(ctx context.Context, key types.NamespacedName) (*dataobject.DataObject, error)
}

type operation struct {
	log      logr.Logger
	client   client.Client
	scheme   *runtime.Scheme
	registry registry.Registry
}

// NewOperation creates a new internal installation operation object.
func NewOperation(log logr.Logger, c client.Client, scheme *runtime.Scheme, registry registry.Registry) Interface {
	return &operation{
		log:      log,
		client:   c,
		scheme:   scheme,
		registry: registry,
	}
}

// Log returns a logging instance
func (o *operation) Log() logr.Logger {
	return o.log
}

// Client returns a controller runtime client.Client
func (o *operation) Client() client.Client {
	return o.client
}

// Schema returns a kubernetes scheme
func (o *operation) Scheme() *runtime.Scheme {
	return o.scheme
}

// Registry returns a registry.Registry instance
func (o *operation) Registry() registry.Registry {
	return o.registry
}

// GetDataObjectFromSecret creates a dataobject from a secret
func (o *operation) GetDataObjectFromSecret(ctx context.Context, key types.NamespacedName) (*dataobject.DataObject, error) {
	secret := &corev1.Secret{}
	if err := o.Client().Get(ctx, key, secret); err != nil {
		return nil, err
	}

	do, err := dataobject.New(secret)
	if err != nil {
		return nil, err
	}

	return do, nil
}
