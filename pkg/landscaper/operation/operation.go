// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package operation

import (
	"github.com/gardener/component-spec/bindings-go/ctf"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	"github.com/gardener/landscaper/pkg/components/model"
)

// RegistriesAccessor is a getter interface for available registries.
type RegistriesAccessor interface {
	// ComponentsRegistry returns a components registry instance.
	ComponentsRegistry() ctf.ComponentResolver
}

// Operation is the type that is used to share common operational data across the landscaper reconciler
type Operation struct {
	scheme            *runtime.Scheme
	eventRecorder     record.EventRecorder
	componentRegistry model.RegistryAccess
}

// NewOperation creates a new internal installation Operation object.
// DEPRECATED: use the Builder instead.
func NewOperation(scheme *runtime.Scheme, recorder record.EventRecorder) *Operation {
	return &Operation{
		scheme:        scheme,
		eventRecorder: recorder,
	}
}

// Copy creates a new operation with the same client, scheme and component resolver
func (o *Operation) Copy() *Operation {
	return &Operation{
		scheme:            o.scheme,
		eventRecorder:     o.eventRecorder,
		componentRegistry: o.componentRegistry,
	}
}

// Scheme returns a kubernetes scheme
func (o *Operation) Scheme() *runtime.Scheme {
	return o.scheme
}

// EventRecorder returns an event recorder to create events.
func (o *Operation) EventRecorder() record.EventRecorder {
	return o.eventRecorder
}

// ComponentsRegistry returns a component registry
func (o *Operation) ComponentsRegistry() model.RegistryAccess {
	return o.componentRegistry
}

// SetComponentsRegistry injects a component blueprintsRegistry into the operation
func (o *Operation) SetComponentsRegistry(registry model.RegistryAccess) *Operation {
	o.componentRegistry = registry
	return o
}
