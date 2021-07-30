// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package resourcemanager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	apimacherrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"

	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	lserrors "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// ManifestApplierOptions describes options for the manifest applier
type ManifestApplierOptions struct {
	Decoder          runtime.Decoder
	KubeClient       client.Client
	DefaultNamespace string

	DeployItemName   string
	DeleteTimeout    time.Duration
	UpdateStrategy   manifestv1alpha2.UpdateStrategy
	Manifests        []managedresource.Manifest
	ManagedResources managedresource.ManagedResourceStatusList
}

// ManifestApplier creates or updated manifest based on their definition.
type ManifestApplier struct {
	mux              sync.Mutex
	log              logr.Logger
	decoder          runtime.Decoder
	kubeClient       client.Client
	defaultNamespace string

	deployItemName   string
	deleteTimeout    time.Duration
	updateStrategy   manifestv1alpha2.UpdateStrategy
	manifests        []managedresource.Manifest
	managedResources managedresource.ManagedResourceStatusList
	managedObjects   []*unstructured.Unstructured
}

// NewManifestApplier creates a new manifest deployer
func NewManifestApplier(log logr.Logger, opts ManifestApplierOptions) *ManifestApplier {
	return &ManifestApplier{
		mux:              sync.Mutex{},
		log:              log,
		decoder:          opts.Decoder,
		kubeClient:       opts.KubeClient,
		defaultNamespace: opts.DefaultNamespace,
		deployItemName:   opts.DeployItemName,
		deleteTimeout:    opts.DeleteTimeout,
		updateStrategy:   opts.UpdateStrategy,
		manifests:        opts.Manifests,
		managedResources: opts.ManagedResources,
	}
}

// GetManagedResourcesStatus returns the managed resources of the applier.
func (a *ManifestApplier) GetManagedResourcesStatus() managedresource.ManagedResourceStatusList {
	return a.managedResources
}

// GetManagedObjects returns all managed objects as unstructured object.
func (a *ManifestApplier) GetManagedObjects() []*unstructured.Unstructured {
	return a.managedObjects
}

// Apply creates or updates all configured manifests.
func (a *ManifestApplier) Apply(ctx context.Context) error {
	var (
		allErrs []error
		errMux  sync.Mutex
		wg      sync.WaitGroup
	)
	// Keep track of the current managed resources before applying so
	// we can then compare which one need to be cleaned up.
	oldManagedResources := a.managedResources
	a.managedResources = make(managedresource.ManagedResourceStatusList, 0)
	for i, m := range a.manifests {
		wg.Add(1)
		go func(i int, m managedresource.Manifest) {
			defer wg.Done()
			if err := a.ApplyObject(ctx, i, m); err != nil {
				errMux.Lock()
				defer errMux.Unlock()
				allErrs = append(allErrs, err)
			}
		}(i, m)
	}
	wg.Wait()
	if len(allErrs) != 0 {
		aggErr := apimacherrors.NewAggregate(allErrs)
		return lserrors.NewWrappedError(apimacherrors.NewAggregate(allErrs),
			"ApplyObjects", "ApplyNewObject", aggErr.Error())
	}

	// remove old objects
	if err := a.cleanupOrphanedResources(ctx, oldManagedResources); err != nil {
		err = fmt.Errorf("unable to cleanup orphaned resources: %w", err)
		return lserrors.NewWrappedError(err,
			"ApplyObjects", "CleanupOrphanedObects", err.Error())
	}
	return nil
}

// ApplyObject applies a managed resource to the target cluster.
func (a *ManifestApplier) ApplyObject(ctx context.Context, i int, manifestData managedresource.Manifest) error {
	if manifestData.Policy == managedresource.IgnorePolicy {
		return nil
	}
	obj := &unstructured.Unstructured{}
	if _, _, err := a.decoder.Decode(manifestData.Manifest.Raw, nil, obj); err != nil {
		return fmt.Errorf("error while decoding manifest at index %d: %w", i, err)
	}

	if len(a.defaultNamespace) != 0 && len(obj.GetNamespace()) == 0 {
		// need to default the namespace if it is not given, as some helmcharts
		// do not use ".Release.Namespace" and depend on the helm/kubectl defaulting.
		// todo: check for clusterwide resources
		obj.SetNamespace(a.defaultNamespace)
	}

	mr := managedresource.ManagedResourceStatus{
		Policy:   manifestData.Policy,
		Resource: *kutil.TypedObjectReferenceFromUnstructuredObject(obj),
	}

	currObj := unstructured.Unstructured{} // can't use obj.NewEmptyInstance() as this returns a runtime.Unstructured object which doesn't implement client.Object
	currObj.GetObjectKind().SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	key := kutil.ObjectKey(obj.GetName(), obj.GetNamespace())
	if err := a.kubeClient.Get(ctx, key, &currObj); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("unable to get object: %w", err)
		}
		// inject manifest specific labels
		kutil.SetMetaDataLabel(obj, manifestv1alpha2.ManagedDeployItemLabel, a.deployItemName)
		if err := a.kubeClient.Create(ctx, obj); err != nil {
			return fmt.Errorf("unable to create resource %s: %w", key.String(), err)
		}
		a.mux.Lock()
		a.managedResources = append(a.managedResources, mr)
		a.managedObjects = append(a.managedObjects, obj)
		a.mux.Unlock()
		return nil
	}

	a.mux.Lock()
	a.managedResources = append(a.managedResources, mr)
	a.managedObjects = append(a.managedObjects, obj)
	a.mux.Unlock()

	// if fallback policy is set and the resource is already managed by another deployer
	// we are not allowed to manage that resource
	if manifestData.Policy == managedresource.FallbackPolicy && !kutil.HasLabelWithValue(obj, manifestv1alpha2.ManagedDeployItemLabel, a.deployItemName) {
		a.log.Info("resource is already managed", "resource", key.String())
		return nil
	}
	// inject manifest specific labels
	kutil.SetMetaDataLabel(obj, manifestv1alpha2.ManagedDeployItemLabel, a.deployItemName)

	// Set the required and immutable fields from the current object.
	// Update fails if these fields are missing
	if err := kutil.SetRequiredNestedFieldsFromObj(&currObj, obj); err != nil {
		return err
	}

	switch a.updateStrategy {
	case manifestv1alpha2.UpdateStrategyUpdate:
		if err := a.kubeClient.Update(ctx, obj); err != nil {
			return fmt.Errorf("unable to update resource %s: %w", key.String(), err)
		}
	case manifestv1alpha2.UpdateStrategyPatch:
		if err := a.kubeClient.Patch(ctx, obj, client.MergeFrom(&currObj)); err != nil {
			return fmt.Errorf("unable to patch resource %s: %w", key.String(), err)
		}
	default:
		return fmt.Errorf("%s is not a valid update strategy", a.updateStrategy)
	}
	return nil
}

// cleanupOrphanedResources removes all managed resources that are not rendered anymore.
func (a *ManifestApplier) cleanupOrphanedResources(ctx context.Context, managedResources []managedresource.ManagedResourceStatus) error {
	var (
		allErrs []error
		wg      sync.WaitGroup
	)

	for _, mr := range managedResources {
		if mr.Policy == managedresource.IgnorePolicy || mr.Policy == managedresource.KeepPolicy {
			continue
		}
		ref := mr.Resource
		obj := kutil.ObjectFromTypedObjectReference(&ref)
		if err := a.kubeClient.Get(ctx, kutil.ObjectKey(ref.Name, ref.Namespace), obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("unable to get object %s %s: %w", obj.GroupVersionKind().String(), obj.GetName(), err)
		}

		if !containsUnstructuredObject(obj, a.managedObjects) {
			wg.Add(1)
			go func(obj *unstructured.Unstructured) {
				defer wg.Done()
				// Delete object and ensure it is actually deleted from the cluster.
				err := kutil.DeleteAndWaitForObjectDeleted(ctx, a.kubeClient, a.deleteTimeout, obj)
				if err != nil {
					allErrs = append(allErrs, err)
				}
			}(obj)
		}
	}
	wg.Wait()

	if len(allErrs) == 0 {
		return nil
	}
	return apimacherrors.NewAggregate(allErrs)
}

func containsUnstructuredObject(obj *unstructured.Unstructured, objects []*unstructured.Unstructured) bool {
	for _, found := range objects {
		if len(obj.GetUID()) != 0 && len(found.GetUID()) != 0 {
			if obj.GetUID() == found.GetUID() {
				return true
			}
			continue
		}
		// todo: check for conversions .e.g. networking.k8s.io -> apps.k8s.io
		if found.GetObjectKind().GroupVersionKind().GroupKind() != obj.GetObjectKind().GroupVersionKind().GroupKind() {
			continue
		}
		if found.GetName() == obj.GetName() && found.GetNamespace() == obj.GetNamespace() {
			return true
		}
	}
	return false
}
