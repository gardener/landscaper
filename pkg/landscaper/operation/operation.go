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

// Interface is the Operation interface that is used to share common operational data across the landscaper reconciler
type Interface interface {
	Log() logr.Logger
	Client() client.Client
	DirectReader() client.Reader
	Scheme() *runtime.Scheme
	RegistriesAccessor
}

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
func NewOperation(log logr.Logger, c client.Client, scheme *runtime.Scheme, compRegistry ctf.ComponentResolver) Interface {
	return &Operation{
		log:               log,
		client:            c,
		scheme:            scheme,
		componentRegistry: compRegistry,
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

// InjectClient injects a kubernetes client into the operation
func (o *Operation) InjectClient(c client.Client) error {
	o.client = c
	return nil
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

// InjectComponentsRegistry injects a component blueprintsRegistry into the operation
func (o *Operation) InjectComponentsRegistry(c ctf.ComponentResolver) error {
	o.componentRegistry = c
	return nil
}

// ComponentRegistryInjector is an interface definition to inject a component registry
type ComponentRegistryInjector interface {
	InjectComponentsRegistry(c ctf.ComponentResolver) error
}

// InjectComponentsRegistryInto is a helper function that tries to inject a component resolver into a struct with a component registry interface
func InjectComponentsRegistryInto(object interface{}, c ctf.ComponentResolver) error {
	if inj, ok := object.(ComponentRegistryInjector); ok {
		return inj.InjectComponentsRegistry(c)
	}
	return nil
}
