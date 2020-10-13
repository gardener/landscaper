// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package operation

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	blueprintsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/blueprints"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	"github.com/gardener/landscaper/pkg/utils/oci"
)

// Operation is the Operation interface that is used to share common operational data across the landscaper reconciler.
type Interface interface {
	Log() logr.Logger
	Client() client.Client
	DirectReader() client.Reader
	Scheme() *runtime.Scheme
	RegistriesAccessor
	// OCIClient returns a oci client to interact with various oci registries.
	OCIClient() oci.Client
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
	// InjectUniversalOCIClient is used to inject a oci client.
	InjectUniversalOCIClient(oci.Client) error
	// CreateOrUpdate creates or updates the given object in the Kubernetes
	// cluster.
	// It uses the internal clients to perform the api calls.
	CreateOrUpdate(ctx context.Context, obj runtime.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error)
}

// RegistriesAccessor is a getter interface for available registries.
type RegistriesAccessor interface {
	// BlueprintsRegistry returns a blueprints blueprintsRegistry instance.
	BlueprintsRegistry() blueprintsregistry.Registry
	// ComponentsRegistry returns a components blueprintsRegistry instance.
	ComponentsRegistry() componentsregistry.Registry
}

type Operation struct {
	log                logr.Logger
	client             client.Client
	directReader       client.Reader
	scheme             *runtime.Scheme
	blueprintsRegistry blueprintsregistry.Registry
	componentRegistry  componentsregistry.Registry
	universalOCIClient oci.Client
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

// OCIClient returns the oci client for the current instance.
func (o *Operation) OCIClient() oci.Client {
	return o.universalOCIClient
}

// InjectUniversalOCIClient injects a oci client into the operation.
func (o *Operation) InjectUniversalOCIClient(c oci.Client) error {
	o.universalOCIClient = c
	return nil
}

// CreateOrUpdate creates or updates the given object in the Kubernetes
// cluster. The object's desired state must be reconciled with the existing
// state inside the passed in callback MutateFn.
//
// The MutateFn is called regardless of creating or updating an object.
//
// It returns the executed operation and an error.
func (o *Operation) CreateOrUpdate(ctx context.Context, obj runtime.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	key, err := client.ObjectKeyFromObject(obj)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	if err := o.DirectReader().Get(ctx, key, obj); err != nil {
		if !errors.IsNotFound(err) {
			return controllerutil.OperationResultNone, err
		}
		if err := kutil.Mutate(f, key, obj); err != nil {
			return controllerutil.OperationResultNone, err
		}
		if err := o.Client().Create(ctx, obj); err != nil {
			return controllerutil.OperationResultNone, err
		}
		return controllerutil.OperationResultCreated, nil
	}

	existing := obj.DeepCopyObject()
	if err := kutil.Mutate(f, key, obj); err != nil {
		return controllerutil.OperationResultNone, err
	}

	if equality.Semantic.DeepEqual(existing, obj) {
		return controllerutil.OperationResultNone, nil
	}

	if err := o.Client().Update(ctx, obj); err != nil {
		return controllerutil.OperationResultNone, err
	}
	return controllerutil.OperationResultUpdated, nil
}
