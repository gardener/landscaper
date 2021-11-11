// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package resourcemanager

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	apimacherrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"

	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	lserrors "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
)

// ApplyManifests creates or updates all configured manifests.
func ApplyManifests(ctx context.Context, log logr.Logger, opts ManifestApplierOptions) (managedresource.ManagedResourceStatusList, error) {
	applier := NewManifestApplier(log, opts)
	if err := applier.Apply(ctx); err != nil {
		return nil, err
	}
	return applier.GetManagedResourcesStatus(), nil
}

// ManifestApplierOptions describes options for the manifest applier
type ManifestApplierOptions struct {
	Decoder          runtime.Decoder
	KubeClient       client.Client
	Clientset        kubernetes.Interface
	DefaultNamespace string

	DeployItemName   string
	DeleteTimeout    time.Duration
	UpdateStrategy   manifestv1alpha2.UpdateStrategy
	Manifests        []managedresource.Manifest
	ManagedResources managedresource.ManagedResourceStatusList
	// Labels defines additional labels that are automatically injected into all resources.
	Labels map[string]string
}

// ManifestApplier creates or updated manifest based on their definition.
type ManifestApplier struct {
	log              logr.Logger
	decoder          runtime.Decoder
	kubeClient       client.Client
	clientset        kubernetes.Interface
	defaultNamespace string

	deployItemName   string
	deleteTimeout    time.Duration
	updateStrategy   manifestv1alpha2.UpdateStrategy
	manifests        []managedresource.Manifest
	managedResources managedresource.ManagedResourceStatusList
	labels           map[string]string

	// properties created during runtime

	// manifestExecutions contains a sorted list of lists of managed resources.
	// The list of list describe execution groups of manifests that can run in parallel.
	//
	// Currently the fist list can be max 3 whereas the first group contains all CRD's.
	// The second group contains all clusterwide resources and teh third one contains all namespaced resources.
	manifestExecutions [3][]*Manifest
	// apiresources is internal cache for api resources where the key is the GroupVersionKind string.
	apiresources map[string]metav1.APIResource
}

const (
	ExecutionGroupCRD = iota
	ExecutionGroupClusterwide
	ExecutionGroupNamespaced
)

// Manifest is the internal representation of a manifest
type Manifest struct {
	TypeMeta metav1.TypeMeta
	// Policy defines the manage policy for that resource.
	Policy managedresource.ManifestPolicy `json:"policy,omitempty"`
	// Manifest defines the raw k8s manifest.
	Manifest *runtime.RawExtension `json:"manifest,omitempty"`
}

// NewManifestApplier creates a new manifest deployer
func NewManifestApplier(log logr.Logger, opts ManifestApplierOptions) *ManifestApplier {
	return &ManifestApplier{
		log:              log,
		decoder:          opts.Decoder,
		kubeClient:       opts.KubeClient,
		clientset:        opts.Clientset,
		defaultNamespace: opts.DefaultNamespace,
		deployItemName:   opts.DeployItemName,
		deleteTimeout:    opts.DeleteTimeout,
		updateStrategy:   opts.UpdateStrategy,
		manifests:        opts.Manifests,
		managedResources: opts.ManagedResources,
		labels:           opts.Labels,

		apiresources: make(map[string]metav1.APIResource),
	}
}

// GetManagedResourcesStatus returns the managed resources of the applier.
func (a *ManifestApplier) GetManagedResourcesStatus() managedresource.ManagedResourceStatusList {
	return a.managedResources
}

// Apply creates or updates all configured manifests.
func (a *ManifestApplier) Apply(ctx context.Context) error {
	if err := a.prepareManifests(); err != nil {
		return err
	}

	var (
		allErrs []error
		errMux  sync.Mutex
	)
	// Keep track of the current managed resources before applying so
	// we can then compare which one need to be cleaned up.
	oldManagedResources := a.managedResources
	a.managedResources = make(managedresource.ManagedResourceStatusList, 0)
	for _, list := range a.manifestExecutions {
		var (
			wg               = sync.WaitGroup{}
			managedResources = make([]managedresource.ManagedResourceStatus, 0)
			mux              sync.Mutex
		)
		for _, m := range list {
			wg.Add(1)
			go func(m *Manifest) {
				defer wg.Done()
				mr, err := a.applyObject(ctx, m)
				if err != nil {
					errMux.Lock()
					defer errMux.Unlock()
					allErrs = append(allErrs, err)
				}
				if mr != nil {
					mux.Lock()
					managedResources = append(managedResources, *mr)
					mux.Unlock()
				}
			}(m)
		}
		wg.Wait()
		sort.Sort(managesResourceList(managedResources))
		a.managedResources = append(a.managedResources, managedResources...)
	}

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

// applyObject applies a managed resource to the target cluster.
func (a *ManifestApplier) applyObject(ctx context.Context, manifest *Manifest) (*managedresource.ManagedResourceStatus, error) {
	if manifest.Policy == managedresource.IgnorePolicy {
		return nil, nil
	}

	gvk := manifest.TypeMeta.GetObjectKind().GroupVersionKind().String()
	obj := &unstructured.Unstructured{}
	if _, _, err := a.decoder.Decode(manifest.Manifest.Raw, nil, obj); err != nil {
		return nil, fmt.Errorf("error while decoding manifest %s: %w", gvk, err)
	}

	if len(a.defaultNamespace) != 0 && len(obj.GetNamespace()) == 0 {
		// need to default the namespace if it is not given, as some helmcharts
		// do not use ".Release.Namespace" and depend on the helm/kubectl defaulting.
		apiresource, err := a.getApiResource(manifest)
		if err != nil {
			return nil, err
		}
		// only default namespaced resources.
		if apiresource.Namespaced {
			obj.SetNamespace(a.defaultNamespace)
		}
	}

	currObj := unstructured.Unstructured{} // can't use obj.NewEmptyInstance() as this returns a runtime.Unstructured object which doesn't implement client.Object
	currObj.GetObjectKind().SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	key := kutil.ObjectKey(obj.GetName(), obj.GetNamespace())
	if err := a.kubeClient.Get(ctx, key, &currObj); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("unable to get object: %w", err)
		}
		// inject labels
		kutil.SetMetaDataLabel(obj, manifestv1alpha2.ManagedDeployItemLabel, a.deployItemName)
		if err := a.kubeClient.Create(ctx, obj); err != nil {
			return nil, fmt.Errorf("unable to create resource %s: %w", key.String(), err)
		}
		return &managedresource.ManagedResourceStatus{
			Policy:   manifest.Policy,
			Resource: *kutil.CoreObjectReferenceFromUnstructuredObject(obj),
		}, nil
	}

	mr := &managedresource.ManagedResourceStatus{
		Policy:   manifest.Policy,
		Resource: *kutil.CoreObjectReferenceFromUnstructuredObject(&currObj),
	}

	// if fallback policy is set and the resource is already managed by another deployer
	// we are not allowed to manage that resource
	if manifest.Policy == managedresource.FallbackPolicy && !kutil.HasLabelWithValue(obj, manifestv1alpha2.ManagedDeployItemLabel, a.deployItemName) {
		a.log.Info("resource is already managed", "resource", key.String())
		return nil, nil
	}
	// inject manifest specific labels
	a.injectLabels(obj)
	kutil.SetMetaDataLabel(obj, manifestv1alpha2.ManagedDeployItemLabel, a.deployItemName)

	// Set the required and immutable fields from the current object.
	// Update fails if these fields are missing
	if err := kutil.SetRequiredNestedFieldsFromObj(&currObj, obj); err != nil {
		return mr, err
	}

	switch a.updateStrategy {
	case manifestv1alpha2.UpdateStrategyUpdate:
		if err := a.kubeClient.Update(ctx, obj); err != nil {
			return mr, fmt.Errorf("unable to update resource %s: %w", key.String(), err)
		}
	case manifestv1alpha2.UpdateStrategyPatch:
		if err := a.kubeClient.Patch(ctx, obj, client.MergeFrom(&currObj)); err != nil {
			return mr, fmt.Errorf("unable to patch resource %s: %w", key.String(), err)
		}
	default:
		return mr, fmt.Errorf("%s is not a valid update strategy", a.updateStrategy)
	}
	return mr, nil
}

func (a *ManifestApplier) injectLabels(obj client.Object) {
	if len(a.labels) == 0 {
		return
	}
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	for key, val := range a.labels {
		labels[key] = val
	}
	obj.SetLabels(labels)
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
		obj := kutil.ObjectFromCoreObjectReference(&ref)
		if err := a.kubeClient.Get(ctx, kutil.ObjectKey(ref.Name, ref.Namespace), obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("unable to get object %s %s: %w", obj.GroupVersionKind().String(), obj.GetName(), err)
		}

		if !containsObjectRef(ref, a.managedResources) {
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

func (a *ManifestApplier) getApiResource(manifest *Manifest) (metav1.APIResource, error) {
	gvk := manifest.TypeMeta.GetObjectKind().GroupVersionKind().String()
	if res, ok := a.apiresources[gvk]; ok {
		return res, nil
	}

	groupVersion := manifest.TypeMeta.GetObjectKind().GroupVersionKind().GroupVersion().String()
	kind := manifest.TypeMeta.GetObjectKind().GroupVersionKind().Kind
	apiresourceList, err := a.clientset.Discovery().ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		return metav1.APIResource{}, fmt.Errorf("unable to get api resources for %s: %w", groupVersion, err)
	}

	for _, apiresource := range apiresourceList.APIResources {
		if apiresource.Kind == kind {
			return apiresource, nil
		}
	}
	return metav1.APIResource{}, fmt.Errorf("unable to get apiresource for %s", gvk)
}

// prepareManifests sorts all manifests.
func (a *ManifestApplier) prepareManifests() error {
	a.manifestExecutions = [3][]*Manifest{}
	for _, obj := range a.manifests {
		typeMeta := metav1.TypeMeta{}
		if err := json.Unmarshal(obj.Manifest.Raw, &typeMeta); err != nil {
			return fmt.Errorf("unable to parse type metadata: %w", err)
		}
		kind := typeMeta.GetObjectKind().GroupVersionKind().Kind

		manifest := &Manifest{
			TypeMeta: typeMeta,
			Policy:   obj.Policy,
			Manifest: obj.Manifest,
		}
		// add to specific execution group
		if kind == "CustomResourceDefinition" {
			a.manifestExecutions[ExecutionGroupCRD] = append(a.manifestExecutions[ExecutionGroupCRD], manifest)
		} else {
			apiresource, err := a.getApiResource(manifest)
			if err != nil {
				return err
			}
			if apiresource.Namespaced {
				a.manifestExecutions[ExecutionGroupNamespaced] = append(a.manifestExecutions[ExecutionGroupNamespaced], manifest)
			} else {
				a.manifestExecutions[ExecutionGroupClusterwide] = append(a.manifestExecutions[ExecutionGroupClusterwide], manifest)
			}
		}
	}

	return nil
}

type managesResourceList []managedresource.ManagedResourceStatus

func (m managesResourceList) Len() int {
	return len(m)
}

func (m managesResourceList) Less(i, j int) bool {
	gvkI := m[i].Resource.String()
	gvkJ := m[j].Resource.String()
	return gvkI < gvkJ
}

func (m managesResourceList) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func containsObjectRef(obj corev1.ObjectReference, objects []managedresource.ManagedResourceStatus) bool {
	for _, mr := range objects {
		found := mr.Resource
		if len(obj.UID) != 0 && len(found.UID) != 0 {
			if obj.UID == found.UID {
				return true
			}
		}
		// todo: check for conversions .e.g. networking.k8s.io -> apps.k8s.io
		if found.GetObjectKind().GroupVersionKind().GroupKind() != obj.GetObjectKind().GroupVersionKind().GroupKind() {
			continue
		}
		if found.Name == obj.Name && found.Namespace == obj.Namespace {
			return true
		}
	}
	return false
}
