// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifest

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

	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/apis/deployer/manifest"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// ObjectApplier creates or updated manifest based on their definition.
type ObjectApplier struct {
	mux        sync.Mutex
	log        logr.Logger
	decoder    runtime.Decoder
	kubeClient client.Client

	deployItemName   string
	deleteTimeout    string
	updateStrategy   manifest.UpdateStrategy
	manifests        []manifest.Manifest
	managedResources []manifest.ManagedResourceStatus
	managedObjects   []*unstructured.Unstructured
}

// Apply creates or updates all configured manifests.
func (a *ObjectApplier) Apply(ctx context.Context) error {
	var (
		allErrs []error
		errMux  sync.Mutex
		wg      sync.WaitGroup
	)
	// Keep track of the current managed resources before applying so
	// we can then compare which one need to be cleaned up.
	oldManagedResources := a.managedResources
	a.managedResources = make([]manifest.ManagedResourceStatus, 0)
	for i, m := range a.manifests {
		wg.Add(1)
		go func(i int, m manifest.Manifest) {
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
		return lsv1alpha1helper.NewWrappedError(apimacherrors.NewAggregate(allErrs),
			"ApplyObjects", "ApplyNewObject", aggErr.Error())
	}

	// remove old objects
	if err := a.cleanupOrphanedResources(ctx, oldManagedResources); err != nil {
		err = fmt.Errorf("unable to cleanup orphaned resources: %w", err)
		return lsv1alpha1helper.NewWrappedError(err,
			"ApplyObjects", "CleanupOrphanedObects", err.Error())
	}
	return nil
}

// ApplyObject applies a managed resource to the target cluster.
func (a *ObjectApplier) ApplyObject(ctx context.Context, i int, manifestData manifest.Manifest) error {
	if manifestData.Policy == manifest.IgnorePolicy {
		return nil
	}
	obj := &unstructured.Unstructured{}
	if _, _, err := a.decoder.Decode(manifestData.Manifest.Raw, nil, obj); err != nil {
		return fmt.Errorf("error while decoding manifest at index %d: %w", i, err)
	}

	mr := manifest.ManagedResourceStatus{
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
	if manifestData.Policy == manifest.FallbackPolicy && !kutil.HasLabelWithValue(obj, manifestv1alpha2.ManagedDeployItemLabel, a.deployItemName) {
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
	case manifest.UpdateStrategyUpdate:
		if err := a.kubeClient.Update(ctx, obj); err != nil {
			return fmt.Errorf("unable to update resource %s: %w", key.String(), err)
		}
	case manifest.UpdateStrategyPatch:
		if err := a.kubeClient.Patch(ctx, &currObj, client.MergeFrom(obj)); err != nil {
			return fmt.Errorf("unable to patch resource %s: %w", key.String(), err)
		}
	default:
		return fmt.Errorf("%s is not a valid update strategy", a.updateStrategy)
	}
	return nil
}

// cleanupOrphanedResources removes all managed resources that are not rendered anymore.
func (a *ObjectApplier) cleanupOrphanedResources(ctx context.Context, mr []manifest.ManagedResourceStatus) error {
	var (
		allErrs []error
		wg      sync.WaitGroup
	)

	for _, mr := range mr {
		if mr.Policy == manifest.IgnorePolicy || mr.Policy == manifest.KeepPolicy {
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
				timeout, _ := time.ParseDuration(a.deleteTimeout)
				err := kutil.DeleteAndWaitForObjectDeleted(ctx, a.kubeClient, timeout, obj)
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
