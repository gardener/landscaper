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

	corev1 "k8s.io/api/core/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	apischema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	apimacherrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"

	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	lserrors "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
)

// ApplyManifests creates or updates all configured manifests.
func ApplyManifests(ctx context.Context, opts ManifestApplierOptions) (managedresource.ManagedResourceStatusList, error) {
	applier := NewManifestApplier(opts)
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
	decoder          runtime.Decoder
	kubeClient       client.Client
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
	apiResourceHandler *ApiResourceHandler
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
func NewManifestApplier(opts ManifestApplierOptions) *ManifestApplier {
	return &ManifestApplier{
		decoder:            opts.Decoder,
		kubeClient:         opts.KubeClient,
		defaultNamespace:   opts.DefaultNamespace,
		deployItemName:     opts.DeployItemName,
		deleteTimeout:      opts.DeleteTimeout,
		updateStrategy:     opts.UpdateStrategy,
		manifests:          opts.Manifests,
		managedResources:   opts.ManagedResources,
		labels:             opts.Labels,
		apiResourceHandler: CreateApiResourceHandler(opts.Clientset),
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
	logger, ctx := logging.FromContextOrNew(ctx, nil, lc.KeyMethod, "applyObject")
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
		apiresource, err := a.apiResourceHandler.GetApiResource(manifest)
		if err != nil {
			return nil, err
		}
		// only default namespaced resources.
		if apiresource.Namespaced {
			obj.SetNamespace(a.defaultNamespace)
		}
	}

	logger.Debug("Applying manifest", lc.KeyResource, kutil.ObjectKeyFromObject(obj).String(), lc.KeyGroupVersionKind, gvk)

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
		logger.Info("Resource is already managed, skip update", lc.KeyResource, key.String())
		return nil, nil
	}

	if manifest.Policy == managedresource.ImmutablePolicy {
		logger.Info("Resource is immutable, skip update", lc.KeyResource, key.String())
		return mr, nil
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
	logger, ctx := logging.FromContextOrNew(ctx, nil, lc.KeyMethod, "cleanupOrphanedResources")

	for _, mr := range managedResources {
		logger2 := logger.WithValues(lc.KeyResource, types.NamespacedName{Namespace: mr.Resource.Namespace, Name: mr.Resource.Name}.String(), lc.KeyResourceKind, mr.Resource.Kind)
		logger2.Debug("Checking resource")
		if mr.Policy == managedresource.IgnorePolicy || mr.Policy == managedresource.KeepPolicy {
			logger2.Debug("Ignoring resource due to policy", lc.KeyManagedResourcePolicy, string(mr.Policy))
			continue
		}
		ref := mr.Resource
		obj := kutil.ObjectFromCoreObjectReference(&ref)
		if err := a.kubeClient.Get(ctx, kutil.ObjectKey(ref.Name, ref.Namespace), obj); err != nil {
			if apierrors.IsNotFound(err) {
				logger2.Debug("Object not found")
				continue
			}
			return fmt.Errorf("unable to get object %s %s: %w", obj.GroupVersionKind().String(), obj.GetName(), err)
		}

		if !containsObjectRef(ref, a.managedResources) {
			logger2.Debug("Object is orphaned and will be deleted")
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

// crdIdentifier generates an identifier string from a GroupVersionKind object
// The version is ignored in this case, because the information whether the resource is namespaced or not
// does not depend on it.
func crdIdentifier(gvk apischema.GroupVersionKind) string {
	return fmt.Sprintf("%s/%s", gvk.Group, gvk.Kind)
}

// prepareManifests sorts all manifests.
func (a *ManifestApplier) prepareManifests() error {
	a.manifestExecutions = [3][]*Manifest{}
	crdNamespacedInfo := map[string]bool{}
	todo := []*Manifest{}
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
			crd := &extv1.CustomResourceDefinition{}
			if err := json.Unmarshal(obj.Manifest.Raw, crd); err != nil {
				return fmt.Errorf("unable to parse CRD: %w", err)
			}
			id := crdIdentifier(apischema.GroupVersionKind{
				Group: crd.Spec.Group,
				Kind:  crd.Spec.Names.Kind,
			})
			crdNamespacedInfo[id] = (crd.Spec.Scope == extv1.NamespaceScoped)
		} else {
			// save manifests for later
			// whether a resource is namespaced or not can only be determined after all CRDs have been evaluated
			todo = append(todo, manifest)
		}
	}
	for _, manifest := range todo {
		// check whether the resource is
		namespaced := false
		apiresource, err := a.apiResourceHandler.GetApiResource(manifest)
		if err != nil {
			// check if the resource matches a not-yet-applied CRD
			ok := false
			namespaced, ok = crdNamespacedInfo[crdIdentifier(manifest.TypeMeta.GroupVersionKind())]
			if !ok {
				// resource not found
				return err
			}
		} else {
			namespaced = apiresource.Namespaced
		}
		if namespaced {
			a.manifestExecutions[ExecutionGroupNamespaced] = append(a.manifestExecutions[ExecutionGroupNamespaced], manifest)
		} else {
			a.manifestExecutions[ExecutionGroupClusterwide] = append(a.manifestExecutions[ExecutionGroupClusterwide], manifest)
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
