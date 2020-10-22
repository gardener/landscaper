// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package operation

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	artifactsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/artifacts"
	blueprintsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/blueprints"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
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
	// InjectRegistry is used to inject a blueprint blueprintsRegistry.
	InjectBlueprintsRegistry(blueprintsregistry.Registry) error
	// InjectComponentRepository is used to inject a component repository.
	InjectComponentsRegistry(componentsregistry.Registry) error
	// InjectArtifactsRegistry is used to inject a artifact repository.
	InjectArtifactsRegistry(artifactsregistry.Registry) error
}

// RegistriesAccessor is a getter interface for available registries.
type RegistriesAccessor interface {
	// BlueprintsRegistry returns a blueprints registry instance.
	BlueprintsRegistry() blueprintsregistry.Registry
	// ComponentsRegistry returns a components registry instance.
	ComponentsRegistry() componentsregistry.Registry
	// ArtifactsRegistry returns a artifacts registry instance.
	ArtifactsRegistry() artifactsregistry.Registry
}

type Operation struct {
	log                logr.Logger
	client             client.Client
	directReader       client.Reader
	scheme             *runtime.Scheme
	blueprintsRegistry blueprintsregistry.Registry
	componentRegistry  componentsregistry.Registry
	artifactRegistry   artifactsregistry.Registry
}

// NewOperation creates a new internal installation Operation object.
func NewOperation(log logr.Logger, c client.Client, scheme *runtime.Scheme, blueRegistry blueprintsregistry.Registry, compRegistry componentsregistry.Registry) Interface {
	return &Operation{
		log:                log,
		client:             c,
		scheme:             scheme,
		blueprintsRegistry: blueRegistry,
		componentRegistry:  compRegistry,
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

// InjectAPIReader injects a readonyl kubernetes client into the operation.
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

// Registry returns a Registry instance
func (o *Operation) BlueprintsRegistry() blueprintsregistry.Registry {
	return o.blueprintsRegistry
}

// InjectRegistry injects a Registry into the actuator
func (o *Operation) InjectBlueprintsRegistry(r blueprintsregistry.Registry) error {
	o.blueprintsRegistry = r
	return nil
}

// ComponentRepository returns a component blueprintsRegistry instance
func (o *Operation) ComponentsRegistry() componentsregistry.Registry {
	return o.componentRegistry
}

// InjectComponentRepository injects a component blueprintsRegistry into the operation
func (o *Operation) InjectComponentsRegistry(c componentsregistry.Registry) error {
	o.componentRegistry = c
	return nil
}

// ComponentRepository returns a component blueprintsRegistry instance
func (o *Operation) ArtifactsRegistry() artifactsregistry.Registry {
	return o.artifactRegistry
}

// InjectComponentRepository injects a component blueprintsRegistry into the operation
func (o *Operation) InjectArtifactsRegistry(r artifactsregistry.Registry) error {
	o.artifactRegistry = r
	return nil
}
