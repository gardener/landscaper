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

	"dario.cat/mergo"
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

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	lserrors "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/deployer/lib/interruption"
	"github.com/gardener/landscaper/pkg/deployer/lib/timeout"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

const (
	TimeoutCheckpointDeployerCleanupOrphaned                 = "deployer: cleanup orphaned"
	TimeoutCheckpointDeployerProcessManagedResourceManifests = "deployer: process managed resource manifests"
	TimeoutCheckpointDeployerProcessManifests                = "deployer: process manifests"
	TimeoutCheckpointDeployerApplyManifests                  = "deployer: apply manifests"
)

// ApplyManifests creates or updates all configured manifests.
func ApplyManifests(ctx context.Context, opts ManifestApplierOptions) (managedresource.ManagedResourceStatusList, error) {
	opts.InterruptionChecker = interruption.NewIgnoreInterruptionChecker()
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
	DeployItem       *lsv1alpha1.DeployItem
	UpdateStrategy   manifestv1alpha2.UpdateStrategy
	Manifests        []managedresource.Manifest
	ManagedResources managedresource.ManagedResourceStatusList
	// Labels defines additional labels that are automatically injected into all resources.
	Labels                     map[string]string
	DeletionGroupsDuringUpdate []managedresource.DeletionGroupDefinition
	InterruptionChecker        interruption.InterruptionChecker
}

// ManifestApplier creates or updated manifest based on their definition.
type ManifestApplier struct {
	decoder          runtime.Decoder
	kubeClient       client.Client
	defaultNamespace string

	deployItemName             string
	deployItem                 *lsv1alpha1.DeployItem
	updateStrategy             manifestv1alpha2.UpdateStrategy
	manifests                  []managedresource.Manifest
	managedResources           managedresource.ManagedResourceStatusList
	labels                     map[string]string
	deletionGroupsDuringUpdate []managedresource.DeletionGroupDefinition
	interruptionChecker        interruption.InterruptionChecker

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
	// AnnotateBeforeCreate defines annotations that are being set before the manifest is being created.
	AnnotateBeforeCreate map[string]string `json:"annotateBeforeCreate,omitempty"`
	// AnnotateBeforeDelete defines annotations that are being set before the manifest is being deleted.
	AnnotateBeforeDelete map[string]string `json:"annotateBeforeDelete,omitempty"`
}

// NewManifestApplier creates a new manifest deployer
func NewManifestApplier(opts ManifestApplierOptions) *ManifestApplier {
	return &ManifestApplier{
		decoder:                    opts.Decoder,
		kubeClient:                 opts.KubeClient,
		defaultNamespace:           opts.DefaultNamespace,
		deployItem:                 opts.DeployItem,
		deployItemName:             opts.DeployItemName,
		updateStrategy:             opts.UpdateStrategy,
		manifests:                  opts.Manifests,
		managedResources:           opts.ManagedResources,
		labels:                     opts.Labels,
		deletionGroupsDuringUpdate: opts.DeletionGroupsDuringUpdate,
		interruptionChecker:        opts.InterruptionChecker,
		apiResourceHandler:         CreateApiResourceHandler(opts.Clientset),
	}
}

// GetManagedResourcesStatus returns the managed resources of the applier.
func (a *ManifestApplier) GetManagedResourcesStatus() managedresource.ManagedResourceStatusList {
	return a.managedResources
}

// Apply creates or updates all configured manifests.
func (a *ManifestApplier) Apply(ctx context.Context) error {
	if err := a.prepareManifests(ctx); err != nil {
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

	var timeoutErr lserrors.LsError
	for _, list := range a.manifestExecutions {
		var (
			wg               = sync.WaitGroup{}
			managedResources = make([]managedresource.ManagedResourceStatus, 0)
			mux              sync.Mutex
		)
		for _, m := range list {

			if _, timeoutErr = timeout.TimeoutExceeded(ctx, a.deployItem, TimeoutCheckpointDeployerApplyManifests); timeoutErr != nil {
				break
			}

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

		if timeoutErr != nil {
			return timeoutErr
		}

		sort.Sort(managesResourceList(managedResources))
		a.managedResources = append(a.managedResources, managedResources...)
	}

	if len(allErrs) != 0 {
		aggErr := apimacherrors.NewAggregate(allErrs)
		return lserrors.NewWrappedError(apimacherrors.NewAggregate(allErrs),
			"ApplyObjects", "ApplyNewObject", aggErr.Error())
	}

	// remove old objects
	if err := a.cleanupOrphanedResourcesInGroups(ctx, oldManagedResources); err != nil {
		err = fmt.Errorf("unable to cleanup orphaned resources: %w", err)
		return lserrors.NewWrappedError(err,
			"ApplyObjects", "cleanupOrphanedResourcesInGroups", err.Error())
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

	if err := read_write_layer.GetUnstructured(ctx, a.kubeClient, key, &currObj, read_write_layer.R000048); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("unable to get object: %w", err)
		}
		// inject labels
		a.injectLabels(obj)
		kutil.SetMetaDataLabel(obj, manifestv1alpha2.ManagedDeployItemLabel, a.deployItemName)

		if manifest.AnnotateBeforeCreate != nil {
			objAnnotations := obj.GetAnnotations()
			if objAnnotations == nil {
				objAnnotations = manifest.AnnotateBeforeCreate
			} else {
				if err := mergo.Merge(&objAnnotations, manifest.AnnotateBeforeCreate, mergo.WithOverride); err != nil {
					return nil, fmt.Errorf("unable to set annotations before create for resource %s: %w", key.String(), err)
				}
			}

			obj.SetAnnotations(objAnnotations)
		}

		if err := a.kubeClient.Create(ctx, obj); err != nil {
			return nil, fmt.Errorf("unable to create resource %s: %w", key.String(), err)
		}
		return &managedresource.ManagedResourceStatus{
			AnnotateBeforeDelete: manifest.AnnotateBeforeDelete,
			Policy:               manifest.Policy,
			Resource:             *kutil.CoreObjectReferenceFromUnstructuredObject(obj),
		}, nil
	}

	mr := &managedresource.ManagedResourceStatus{
		AnnotateBeforeDelete: manifest.AnnotateBeforeDelete,
		Policy:               manifest.Policy,
		Resource:             *kutil.CoreObjectReferenceFromUnstructuredObject(&currObj),
	}

	// if fallback policy is set and the resource is already managed by another deployer
	// we are not allowed to manage that resource
	if manifest.Policy == managedresource.FallbackPolicy && !kutil.HasLabelWithValue(&currObj, manifestv1alpha2.ManagedDeployItemLabel, a.deployItemName) {
		logger.Info("Resource is already managed, skip update", lc.KeyResource, key.String())
		return nil, nil
	}

	if manifest.Policy == managedresource.ImmutablePolicy {
		logger.Info("Resource is immutable, skip update", lc.KeyResource, key.String())
		return mr, nil
	}

	switch a.updateStrategy {
	case manifestv1alpha2.UpdateStrategyUpdate:
		fallthrough
	case manifestv1alpha2.UpdateStrategyPatch:
		// inject manifest specific labels
		a.injectLabels(obj)
		kutil.SetMetaDataLabel(obj, manifestv1alpha2.ManagedDeployItemLabel, a.deployItemName)

		// Set the required and immutable fields from the current object.
		// Update fails if these fields are missing
		if err := kutil.SetRequiredNestedFieldsFromObj(&currObj, obj); err != nil {
			return mr, err
		}

		if a.updateStrategy == manifestv1alpha2.UpdateStrategyUpdate {
			if err := a.kubeClient.Update(ctx, obj); err != nil {
				return mr, fmt.Errorf("unable to update resource %s: %w", key.String(), err)
			}
		} else {
			if err := a.kubeClient.Patch(ctx, obj, client.MergeFrom(&currObj)); err != nil {
				return mr, fmt.Errorf("unable to patch resource %s: %w", key.String(), err)
			}
		}
	case manifestv1alpha2.UpdateStrategyMerge:
		fallthrough
	case manifestv1alpha2.UpdateStrategyMergeOverwrite:
		var mergeOpts []func(*mergo.Config)
		if a.updateStrategy == manifestv1alpha2.UpdateStrategyMergeOverwrite {
			mergeOpts = []func(*mergo.Config){
				mergo.WithOverride,
			}
		} else {
			mergeOpts = []func(*mergo.Config){}
		}

		if err := mergo.Merge(&currObj.Object, obj.Object, mergeOpts...); err != nil {
			return mr, fmt.Errorf("unable to merge changes for resource %s: %w", key.String(), err)
		}

		// inject manifest specific labels
		a.injectLabels(&currObj)
		kutil.SetMetaDataLabel(&currObj, manifestv1alpha2.ManagedDeployItemLabel, a.deployItemName)

		if err := a.kubeClient.Update(ctx, &currObj); err != nil {
			return mr, fmt.Errorf("unable to update resource %s: %w", key.String(), err)
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

func (a *ManifestApplier) cleanupOrphanedResourcesInGroups(ctx context.Context, oldManagedResources []managedresource.ManagedResourceStatus) error {
	logger, ctx := logging.FromContextOrNew(ctx, nil, lc.KeyMethod, "cleanupOrphanedResourcesInGroups")
	orphanedManagedResources := []managedresource.ManagedResourceStatus{}

	_, err := timeout.TimeoutExceeded(ctx, a.deployItem, TimeoutCheckpointDeployerCleanupOrphaned)
	if err != nil {
		return err
	}

	for i := range oldManagedResources {
		mr := &oldManagedResources[i]

		mrLogger, mrCtx := logger.WithValuesAndContext(ctx,
			lc.KeyResource, types.NamespacedName{Namespace: mr.Resource.Namespace, Name: mr.Resource.Name}.String(),
			lc.KeyResourceKind, mr.Resource.Kind)
		mrLogger.Debug("Checking resource")

		ok, err := FilterByPolicy(mrCtx, mr, a.kubeClient, a.deployItemName)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}

		ok = a.filterOrphaned(mr)
		if !ok {
			continue
		}

		mrLogger.Debug("Object is orphaned and will be deleted")
		orphanedManagedResources = append(orphanedManagedResources, *mr)
	}

	return DeleteManagedResources(
		ctx,
		orphanedManagedResources,
		a.deletionGroupsDuringUpdate,
		a.kubeClient,
		a.deployItem,
		a.interruptionChecker,
	)
}

// FilterByPolicy is used during the deletion of manifest deployitems and manifest-only helm deployitems.
// It returns true if the deployitem can be deleted according to its policy, and false if it must not be deleted.
func FilterByPolicy(ctx context.Context, mr *managedresource.ManagedResourceStatus, targetClient client.Client, deployItemName string) (bool, error) {
	logger, ctx := logging.FromContextOrNew(ctx, nil)

	if mr.Policy == managedresource.IgnorePolicy || mr.Policy == managedresource.KeepPolicy {
		logger.Debug("Ignoring resource due to policy", lc.KeyManagedResourcePolicy, string(mr.Policy))
		return false, nil
	}

	if mr.Policy == managedresource.FallbackPolicy {
		// if fallback policy is set and the resource is already managed by another deployer
		// we are not allowed to manage that resource
		ref := mr.Resource
		obj := kutil.ObjectFromCoreObjectReference(&ref)
		key := kutil.ObjectKey(ref.Name, ref.Namespace)

		if err := read_write_layer.GetUnstructured(ctx, targetClient, key, obj, read_write_layer.R000049); err != nil {
			if apierrors.IsNotFound(err) {
				logger.Debug("Object not found")
				return false, nil
			}
			return false, fmt.Errorf("unable to get object %s %s: %w", obj.GroupVersionKind().String(), obj.GetName(), err)
		}
		if !kutil.HasLabelWithValue(obj, manifestv1alpha2.ManagedDeployItemLabel, deployItemName) {
			logger.Info("Resource is already managed, skip cleanup")
			return false, nil
		}
	}

	return true, nil
}

// filterOrphaned returns true if the resource is orphaned, i.e. not contained in a.managedResources.
func (a *ManifestApplier) filterOrphaned(mr *managedresource.ManagedResourceStatus) bool {
	return !containsObjectRef(mr.Resource, a.managedResources)
}

// crdIdentifier generates an identifier string from a GroupVersionKind object
// The version is ignored in this case, because the information whether the resource is namespaced or not
// does not depend on it.
func crdIdentifier(gvk apischema.GroupVersionKind) string {
	return fmt.Sprintf("%s/%s", gvk.Group, gvk.Kind)
}

// prepareManifests sorts all manifests.
func (a *ManifestApplier) prepareManifests(ctx context.Context) error {
	a.manifestExecutions = [3][]*Manifest{}
	crdNamespacedInfo := map[string]bool{}
	todo := []*Manifest{}

	managedResourceManifests, err := lib.ExpandManagedResourceManifests(a.manifests)
	if err != nil {
		return err
	}

	for _, obj := range managedResourceManifests {

		if _, err := timeout.TimeoutExceeded(ctx, a.deployItem, TimeoutCheckpointDeployerProcessManagedResourceManifests); err != nil {
			return err
		}

		typeMeta := metav1.TypeMeta{}
		if err := json.Unmarshal(obj.Manifest.Raw, &typeMeta); err != nil {
			return fmt.Errorf("unable to parse type metadata: %w", err)
		}
		kind := typeMeta.GetObjectKind().GroupVersionKind().Kind

		manifest := &Manifest{
			TypeMeta:             typeMeta,
			Policy:               obj.Policy,
			Manifest:             obj.Manifest,
			AnnotateBeforeCreate: obj.AnnotateBeforeCreate,
			AnnotateBeforeDelete: obj.AnnotateBeforeDelete,
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
		if _, err := timeout.TimeoutExceeded(ctx, a.deployItem, TimeoutCheckpointDeployerProcessManifests); err != nil {
			return err
		}

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
