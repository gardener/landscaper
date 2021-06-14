// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package operation

import (
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RegistriesAccessor is a getter interface for available registries.
type RegistriesAccessor interface {
	// ComponentsRegistry returns a components registry instance.
	ComponentsRegistry() ctf.ComponentResolver
}

// Operation is the type that is used to share common operational data across the landscaper reconciler
type Operation struct {
	log               logr.Logger
	client            client.Client
	directReader      client.Reader
	scheme            *runtime.Scheme
	componentRegistry ctf.ComponentResolver
}

// NewOperation creates a new internal installation Operation object.
func NewOperation(log logr.Logger, c client.Client, scheme *runtime.Scheme) *Operation {
	return &Operation{
		log:    log,
		client: c,
		scheme: scheme,
	}
}

// Copy creates a new operation with the same client, scheme and component resolver
func (o *Operation) Copy() *Operation {
	return &Operation{
		log:               o.log,
		client:            o.client,
		directReader:      o.directReader,
		scheme:            o.scheme,
		componentRegistry: o.componentRegistry,
	}
}

// Log returns a logging instance
func (o *Operation) Log() logr.Logger {
	return o.log
}

// Client returns a controller runtime client.Registry
func (o *Operation) Client() client.Client {
	return o.client
}

// DirectReader returns a direct readonly api reader.
func (o *Operation) DirectReader() client.Reader {
	if o.directReader == nil {
		return o.client
	}
	return o.directReader
}

// Scheme returns a kubernetes scheme
func (o *Operation) Scheme() *runtime.Scheme {
	return o.scheme
}

// ComponentsRegistry returns a component blueprintsRegistry instance
func (o *Operation) ComponentsRegistry() ctf.ComponentResolver {
	return o.componentRegistry
}

// SetComponentsRegistry injects a component blueprintsRegistry into the operation
func (o *Operation) SetComponentsRegistry(c ctf.ComponentResolver) *Operation {
	o.componentRegistry = c
	return o
}
