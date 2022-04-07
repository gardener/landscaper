// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package operation

import (
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
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
	scheme            *runtime.Scheme
	eventRecorder     record.EventRecorder
	componentRegistry ctf.ComponentResolver
}

// NewOperation creates a new internal installation Operation object.
// DEPRECATED: use the Builder instead.
func NewOperation(log logr.Logger, c client.Client, scheme *runtime.Scheme, recorder record.EventRecorder) *Operation {
	return &Operation{
		log:           log,
		client:        c,
		scheme:        scheme,
		eventRecorder: recorder,
	}
}

// Copy creates a new operation with the same client, scheme and component resolver
func (o *Operation) Copy() *Operation {
	return &Operation{
		log:               o.log,
		client:            o.client,
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

func (o *Operation) Writer() *read_write_layer.Writer {
	return read_write_layer.NewWriter(o.log, o.client)
}

// Scheme returns a kubernetes scheme
func (o *Operation) Scheme() *runtime.Scheme {
	return o.scheme
}

// EventRecorder returns an event recorder to create events.
func (o *Operation) EventRecorder() record.EventRecorder {
	return o.eventRecorder
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
