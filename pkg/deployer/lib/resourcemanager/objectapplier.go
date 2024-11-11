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
	"k8s.io/client-go/rest"
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
	if _, err := applier.Apply(ctx); err != nil {
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

	LsUncachedClient client.Client
	LsRestConfig     *rest.Config
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
	lsUncachedClient           client.Client
	lsRestConfig               *rest.Config

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
	// PatchAfterDeployment defines a patch that is being applied after an object has been deployed.
	PatchAfterDeployment *runtime.RawExtension `json:"patchAfterDeployment,omitempty"`
	// PatchBeforeDelete defines a patch that is being applied before an object is being deleted.
	PatchBeforeDelete *runtime.RawExtension `json:"patchBeforeDelete,omitempty"`
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
		lsUncachedClient:           opts.LsUncachedClient,
		lsRestConfig:               opts.LsRestConfig,
	}
}

// GetManagedResourcesStatus returns the managed resources of the applier.
func (a *ManifestApplier) GetManagedResourcesStatus() managedresource.ManagedResourceStatusList {
	return a.managedResources
}

// Apply creates or updates all configured manifests.
func (a *ManifestApplier) Apply(ctx context.Context) ([]*PatchInfo, error) {
	if err := a.prepareManifests(ctx); err != nil {
		return nil, err
	}

	var (
		allErrs    []error
		errMux     sync.Mutex
		patchInfos = make([]*PatchInfo, 0)
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
				mr, patchInfo, err := a.applyObject(ctx, m)
				if err != nil {
					errMux.Lock()
					defer errMux.Unlock()
					allErrs = append(allErrs, err)
				}
				if mr != nil {
					mux.Lock()
					managedResources = append(managedResources, *mr)
					if patchInfo != nil {
						patchInfos = append(patchInfos, patchInfo)
					}
					mux.Unlock()
				}
			}(m)
		}
		wg.Wait()

		if timeoutErr != nil {
			return nil, timeoutErr
		}

		sort.Sort(managesResourceList(managedResources))
		a.managedResources = append(a.managedResources, managedResources...)
	}

	if len(allErrs) != 0 {
		aggErr := apimacherrors.NewAggregate(allErrs)
		return nil, lserrors.NewWrappedError(apimacherrors.NewAggregate(allErrs), "ApplyObjects", "ApplyNewObject", aggErr.Error())
	}

	// remove old objects
	if err := a.cleanupOrphanedResourcesInGroups(ctx, oldManagedResources); err != nil {
		err = fmt.Errorf("unable to cleanup orphaned resources: %w", err)
		return nil, lserrors.NewWrappedError(err, "ApplyObjects", "cleanupOrphanedResourcesInGroups", err.Error())
	}
	return patchInfos, nil
}

// applyObject applies a managed resource to the target cluster.
func (a *ManifestApplier) applyObject(ctx context.Context, manifest *Manifest) (*managedresource.ManagedResourceStatus, *PatchInfo, error) {
	logger, ctx := logging.FromContextOrNew(ctx, nil, lc.KeyMethod, "applyObject")
	if manifest.Policy == managedresource.IgnorePolicy {
		return nil, nil, nil
	}

	obj, err := a.getUnstructuredManifestObject(ctx, manifest)
	if err != nil {
		return nil, nil, err
	}
	key := client.ObjectKeyFromObject(obj)

	currObj := unstructured.Unstructured{} // can't use obj.NewEmptyInstance() as this returns a runtime.Unstructured object which doesn't implement client.Object
	currObj.GetObjectKind().SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	if err := read_write_layer.GetUnstructured(ctx, a.kubeClient, key, &currObj, read_write_layer.R000048); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, nil, fmt.Errorf("unable to get object: %w", err)
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
					return nil, nil, fmt.Errorf("unable to set annotations before create for resource %s: %w", key.String(), err)
				}
			}

			obj.SetAnnotations(objAnnotations)
		}

		if err := a.kubeClient.Create(ctx, obj); err != nil {
			return nil, nil, fmt.Errorf("unable to create resource %s: %w", key.String(), err)
		}

		var patchInfo *PatchInfo
		if manifest.PatchAfterDeployment != nil {
			patchInfo = &PatchInfo{
				Resource: obj,
				Patch:    manifest.PatchAfterDeployment,
			}
		}

		return &managedresource.ManagedResourceStatus{
			AnnotateBeforeDelete: manifest.AnnotateBeforeDelete,
			PatchBeforeDelete:    manifest.PatchBeforeDelete,
			Policy:               manifest.Policy,
			Resource:             *kutil.CoreObjectReferenceFromUnstructuredObject(obj),
		}, patchInfo, nil
	}

	mr := &managedresource.ManagedResourceStatus{
		AnnotateBeforeDelete: manifest.AnnotateBeforeDelete,
		PatchBeforeDelete:    manifest.PatchBeforeDelete,
		Policy:               manifest.Policy,
		Resource:             *kutil.CoreObjectReferenceFromUnstructuredObject(&currObj),
	}

	// if fallback policy is set and the resource is already managed by another deployer
	// we are not allowed to manage that resource
	if manifest.Policy == managedresource.FallbackPolicy && !kutil.HasLabelWithValue(&currObj, manifestv1alpha2.ManagedDeployItemLabel, a.deployItemName) {
		logger.Info("Resource is already managed, skip update", lc.KeyResource, key.String())
		return nil, nil, nil
	}

	if manifest.Policy == managedresource.ImmutablePolicy {
		logger.Info("Resource is immutable, skip update", lc.KeyResource, key.String())
		return mr, nil, nil
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
			return mr, nil, err
		}

		if a.updateStrategy == manifestv1alpha2.UpdateStrategyUpdate {
			if err := a.kubeClient.Update(ctx, obj); err != nil {
				return mr, nil, fmt.Errorf("unable to update resource %s: %w", key.String(), err)
			}
		} else {
			if err := a.kubeClient.Patch(ctx, obj, client.MergeFrom(&currObj)); err != nil {
				return mr, nil, fmt.Errorf("unable to patch resource %s: %w", key.String(), err)
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
			return mr, nil, fmt.Errorf("unable to merge changes for resource %s: %w", key.String(), err)
		}

		// inject manifest specific labels
		a.injectLabels(&currObj)
		kutil.SetMetaDataLabel(&currObj, manifestv1alpha2.ManagedDeployItemLabel, a.deployItemName)

		if err := a.kubeClient.Update(ctx, &currObj); err != nil {
			return mr, nil, fmt.Errorf("unable to update resource %s: %w", key.String(), err)
		}
	default:
		return mr, nil, fmt.Errorf("%s is not a valid update strategy", a.updateStrategy)
	}

	var patchInfo *PatchInfo
	if manifest.PatchAfterDeployment != nil {
		patchInfo = &PatchInfo{
			Resource: &currObj,
			Patch:    manifest.PatchAfterDeployment,
		}
	}

	return mr, patchInfo, nil
}

func (a *ManifestApplier) getUnstructuredManifestObject(ctx context.Context, manifest *Manifest) (*unstructured.Unstructured, error) {
	logger, _ := logging.FromContextOrNew(ctx, nil, lc.KeyMethod, "getManifestObjectKey")

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

	return obj, nil
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

func (a *ManifestApplier) cleanupOrphanedResourcesInGroups(ctx context.Context,
	oldManagedResources []managedresource.ManagedResourceStatus) error {

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

	for i := range orphanedManagedResources {
		mr := &orphanedManagedResources[i]

		if _, err := AnnotateAndPatchBeforeDelete(ctx, mr, a.kubeClient); err != nil {
			return err
		}
	}

	return DeleteManagedResources(
		ctx,
		a.lsUncachedClient,
		orphanedManagedResources,
		a.deletionGroupsDuringUpdate,
		a.kubeClient,
		a.deployItem,
		a.interruptionChecker,
		a.lsRestConfig,
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

func AnnotateAndPatchBeforeDelete(ctx context.Context, mr *managedresource.ManagedResourceStatus, targetClient client.Client) (notFound bool, err error) {
	if mr.AnnotateBeforeDelete == nil && mr.PatchBeforeDelete == nil {
		return false, nil
	}

	logger, ctx := logging.FromContextOrNew(ctx, nil)

	ref := mr.Resource
	obj := kutil.ObjectFromCoreObjectReference(&ref)
	currObj := unstructured.Unstructured{}
	currObj.GetObjectKind().SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	key := kutil.ObjectKey(obj.GetName(), obj.GetNamespace())
	if err := read_write_layer.GetUnstructured(ctx, targetClient, key, &currObj, read_write_layer.R000052); err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, fmt.Errorf("unable to get resource with before-delete annotations %s %s: %w",
			obj.GroupVersionKind().String(), obj.GetName(), err)
	}

	// annotate before delete
	if mr.AnnotateBeforeDelete != nil {
		objAnnotations := currObj.GetAnnotations()
		if objAnnotations == nil {
			objAnnotations = mr.AnnotateBeforeDelete
		} else {
			if err := mergo.Merge(&objAnnotations, mr.AnnotateBeforeDelete, mergo.WithOverride); err != nil {
				logger.Error(err, "unable to merge resource annotations with before-delete annotations")
				return false, fmt.Errorf("unable to merge resource annotations with before-delete annotations %s %s: %w",
					obj.GroupVersionKind().String(), obj.GetName(), err)
			}
		}
		currObj.SetAnnotations(objAnnotations)
	}

	// patch before delete
	if mr.PatchBeforeDelete != nil {
		patchObj := make(map[string]interface{})
		if err := json.Unmarshal(mr.PatchBeforeDelete.Raw, &patchObj); err != nil {
			return false, fmt.Errorf("error while decoding patch: %w", err)
		}

		if err := mergo.Merge(&currObj.Object, patchObj, mergo.WithOverride); err != nil {
			return false, fmt.Errorf("unable to merge changes changes before delete for resource %s: %w", key.String(), err)
		}
	}

	if err := targetClient.Update(ctx, &currObj); err != nil {
		if apierrors.IsConflict(err) {
			logger.Info("unable to update resource with before-delete annotations due to a conflict", lc.KeyError, err.Error())
			return false, fmt.Errorf("unable to update resource with before-delete annotations due to a conflict %s %s: %w",
				obj.GroupVersionKind().String(), obj.GetName(), err)
		} else {
			logger.Error(err, "unable to update resource with before-delete annotations")
			return false, fmt.Errorf("unable to update resource with before-delete annotations %s %s: %w",
				obj.GroupVersionKind().String(), obj.GetName(), err)
		}
	}

	return false, nil
}

// filterOrphaned returns true if the resource is orphaned, i.e. not contained in a.managedResources.
func (a *ManifestApplier) filterOrphaned(mr *managedresource.ManagedResourceStatus) bool {
	return !containsObjectRef(mr.Resource, a.managedResources)
}

func (a *ManifestApplier) PatchAfterDeployment(ctx context.Context, patchInfos []*PatchInfo) error {
	for i := range patchInfos {
		patchInfo := patchInfos[i]
		key := client.ObjectKeyFromObject(patchInfo.Resource)
		if err := read_write_layer.GetUnstructured(ctx, a.kubeClient, key, patchInfo.Resource, read_write_layer.R000011); err != nil {
			return err
		}

		patchObj := make(map[string]interface{})

		if err := json.Unmarshal(patchInfo.Patch.Raw, &patchObj); err != nil {
			return fmt.Errorf("error while decoding patch: %w", err)
		}

		if err := mergo.Merge(&patchInfo.Resource.Object, patchObj, mergo.WithOverride); err != nil {
			return fmt.Errorf("unable to patch changes for resource %s: %w", key.String(), err)
		}

		if err := a.kubeClient.Update(ctx, patchInfo.Resource); err != nil {
			return fmt.Errorf("unable to patch resource %s: %w", key.String(), err)
		}
	}

	return nil
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
			PatchAfterDeployment: obj.PatchAfterDeployment,
			PatchBeforeDelete:    obj.PatchBeforeDelete,
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

type PatchInfo struct {
	Resource *unstructured.Unstructured
	Patch    *runtime.RawExtension
}
