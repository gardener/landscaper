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
	"github.com/gardener/landscaper/pkg/utils/componentrepository"
)

// Operation is the Operation interface that is used to share common operational data across the landscaper reconciler.
type Interface interface {
	Log() logr.Logger
	Client() client.Client
	Scheme() *runtime.Scheme
	Registry() registry.Registry
	ComponentRepository() componentrepository.Client
	GetDataObjectFromSecret(ctx context.Context, key types.NamespacedName) (*dataobject.DataObject, error)
	// InjectLogger is used to inject Loggers into components that need them
	// and don't otherwise have opinions.
	InjectLogger(l logr.Logger) error
	// InjectClient is used by the ControllerManager to inject client into Sources, EventHandlers, Predicates, and
	// Reconciles
	InjectClient(client.Client) error
	// InjectScheme is used by the ControllerManager to inject Scheme into Sources, EventHandlers, Predicates, and
	// Reconciles
	InjectScheme(scheme *runtime.Scheme) error
	// InjectRegistry is used to inject a blueprint registry.
	InjectRegistry(registry.Registry) error
	// InjectComponentRepository is used to inject a component repository.
	InjectComponentRepository(componentrepository.Client) error
}

type Operation struct {
	log                 logr.Logger
	client              client.Client
	scheme              *runtime.Scheme
	registry            registry.Registry
	componentRepository componentrepository.Client
}

// NewOperation creates a new internal installation Operation object.
func NewOperation(log logr.Logger, c client.Client, scheme *runtime.Scheme, registry registry.Registry, compRepo componentrepository.Client) Interface {
	return &Operation{
		log:                 log,
		client:              c,
		scheme:              scheme,
		registry:            registry,
		componentRepository: compRepo,
	}
}

// Log returns a logging instance
func (o *Operation) Log() logr.Logger {
	return o.log
}

// InjectLogger injects a logger in the opeation
func (o *Operation) InjectLogger(l logr.Logger) error {
	o.log = l
	return nil
}

// Client returns a controller runtime client.Client
func (o *Operation) Client() client.Client {
	return o.client
}

// InjectClient injects a kubernetes client into the operation
func (o *Operation) InjectClient(c client.Client) error {
	o.client = c
	return nil
}

// Schema returns a kubernetes scheme
func (o *Operation) Scheme() *runtime.Scheme {
	return o.scheme
}

// InjectScheme injects the used kubernetes scheme into the operation
func (o *Operation) InjectScheme(scheme *runtime.Scheme) error {
	o.scheme = scheme
	return nil
}

// Registry returns a Registry instance
func (o *Operation) Registry() registry.Registry {
	return o.registry
}

// InjectRegistry injects a Registry into the actuator
func (o *Operation) InjectRegistry(r registry.Registry) error {
	o.registry = r
	return nil
}

// ComponentRepository returns a component registry instance
func (o *Operation) ComponentRepository() componentrepository.Client {
	return o.componentRepository
}

// InjectComponentRepository injects a component registry into the operation
func (o *Operation) InjectComponentRepository(c componentrepository.Client) error {
	o.componentRepository = c
	return nil
}

// GetDataObjectFromSecret creates a dataobject from a secret
func (o *Operation) GetDataObjectFromSecret(ctx context.Context, key types.NamespacedName) (*dataobject.DataObject, error) {
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
