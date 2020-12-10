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

// Operation is the Operation interface that is used to share common operational data across the landscaper reconciler.
type Interface interface {
	Log() logr.Logger
	Client() client.Client
	DirectReader() client.Reader
	Scheme() *runtime.Scheme
	RegistriesAccessor
	// InjectLogger is used to inject Loggers into components that need them
	// and don't otherwise have opinions.
	InjectLogger(l logr.Logger) error
	// InjectClient is used by the ControllerManager to inject client into Sources, EventHandlers, Predicates, and
	// Reconciles
	InjectClient(client.Client) error
	// InjectAPIReader is used by the ControllerManager to inject readonly api client
	InjectAPIReader(r client.Reader) error
	// InjectScheme is used by the ControllerManager to inject Scheme into Sources, EventHandlers, Predicates, and
	// Reconciles
	InjectScheme(scheme *runtime.Scheme) error
}

// RegistriesAccessor is a getter interface for available registries.
type RegistriesAccessor interface {
	// ComponentsRegistry returns a components registry instance.
	ComponentsRegistry() ctf.ComponentResolver
}

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

// InjectLogger injects a logger in the opeation
func (o *Operation) InjectLogger(l logr.Logger) error {
	o.log = l
	return nil
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

// InjectAPIReader injects a readonly kubernetes client into the operation.
func (o *Operation) InjectAPIReader(r client.Reader) error {
	o.directReader = r
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

// ComponentRepository returns a component blueprintsRegistry instance
func (o *Operation) ComponentsRegistry() ctf.ComponentResolver {
	return o.componentRegistry
}

// InjectComponentRepository injects a component blueprintsRegistry into the operation
func (o *Operation) InjectComponentsRegistry(c ctf.ComponentResolver) error {
	o.componentRegistry = c
	return nil
}
