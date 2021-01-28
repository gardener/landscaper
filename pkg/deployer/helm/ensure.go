// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	apimacherrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"
	"github.com/gardener/landscaper/pkg/utils"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	"github.com/gardener/landscaper/pkg/utils/kubernetes/health"
)

// ApplyFiles applies the helm templated files to the target cluster.
func (h *Helm) ApplyFiles(ctx context.Context, files map[string]string, exports map[string]interface{}) error {
	currOp := "ApplyFile"
	_, targetClient, err := h.TargetClient()
	if err != nil {
		return lsv1alpha1helper.NewWrappedError(err,
			currOp, "TargetClusterClient", err.Error())
	}

	objects, err := kutil.ParseFiles(h.log, files)
	if err != nil {
		return lsv1alpha1helper.NewWrappedError(err,
			currOp, "DecodeHelmTemplatedObjects", err.Error())
	}

	if h.ProviderStatus == nil {
		h.ProviderStatus = &helmv1alpha1.ProviderStatus{
			TypeMeta: metav1.TypeMeta{
				APIVersion: helmv1alpha1.SchemeGroupVersion.String(),
				Kind:       "ProviderStatus",
			},
			ManagedResources: make([]lsv1alpha1.TypedObjectReference, 0),
		}
	}

	for _, obj := range objects {
		exports, err = h.addExport(exports, obj)
		if err != nil {
			return lsv1alpha1helper.NewWrappedError(err,
				currOp, "ReadExportValues", err.Error())
		}
		h.injectLabels(obj)
	}

	managedResources := make([]lsv1alpha1.TypedObjectReference, len(objects))
	for i, obj := range objects {
		// need to default the namespace if it is not given, as some helmcharts
		// do not use ".Release.Namespace" and depend on the helm/kubectl defaulting.
		// todo: check for clusterwide resources
		if len(obj.GetNamespace()) == 0 {
			obj.SetNamespace(h.ProviderConfiguration.Namespace)
		}
		if err := h.ApplyObject(ctx, targetClient, obj); err != nil {
			return err
		}

		managedResources[i] = lsv1alpha1.TypedObjectReference{
			APIVersion: obj.GetAPIVersion(),
			Kind:       obj.GetKind(),
			ObjectReference: lsv1alpha1.ObjectReference{
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
			},
		}
	}
	h.ProviderStatus.ManagedResources = managedResources

	if err := h.cleanupOrphanedResources(ctx, targetClient, h.ProviderStatus.ManagedResources, objects); err != nil {
		err = fmt.Errorf("unable to cleanup orphaned resources: %w", err)
		return lsv1alpha1helper.NewWrappedError(err,
			currOp, "CleanupOrphanedResources", err.Error())
	}

	h.DeployItem.Status.ProviderStatus, err = encodeStatus(h.ProviderStatus)
	if err != nil {
		return lsv1alpha1helper.NewWrappedError(err,
			currOp, "ProviderStatus", err.Error())
	}
	if err := h.kubeClient.Status().Update(ctx, h.DeployItem); err != nil {
		return lsv1alpha1helper.NewWrappedError(err,
			currOp, "UpdateStatus", err.Error())
	}

	if err := h.createOrUpdateExport(ctx, exports); err != nil {
		return lsv1alpha1helper.NewWrappedError(err,
			currOp, "CreateExport", err.Error())
	}

	if h.ProviderConfiguration.HealthChecks.DisableDefault {
		h.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
		h.DeployItem.Status.ObservedGeneration = h.DeployItem.Generation
		h.DeployItem.Status.LastError = nil
		return nil
	}

	return h.CheckResourcesHealth(ctx, targetClient)
}

// CheckResourcesHealth checks if the managed resources are Ready/Healthy.
func (h *Helm) CheckResourcesHealth(ctx context.Context, client client.Client) error {
	var (
		currOp = "CheckResourcesHealthHelm"
	)

	if len(h.ProviderStatus.ManagedResources) == 0 {
		return nil
	}

	objects := make([]*unstructured.Unstructured, len(h.ProviderStatus.ManagedResources))
	for i, ref := range h.ProviderStatus.ManagedResources {
		obj := kutil.ObjectFromTypedObjectReference(&ref)
		objects[i] = obj
	}

	timeout, _ := time.ParseDuration(h.ProviderConfiguration.HealthChecks.Timeout)
	if err := health.WaitForObjectsHealthy(ctx, timeout, h.log, client, objects); err != nil {
		return lsv1alpha1helper.NewWrappedError(err,
			currOp, "CheckResourcesReadiness", err.Error())
	}

	h.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
	h.DeployItem.Status.ObservedGeneration = h.DeployItem.Generation
	h.DeployItem.Status.LastError = nil
	return nil
}

func (h *Helm) createOrUpdateExport(ctx context.Context, values map[string]interface{}) error {
	if len(values) == 0 {
		return nil
	}
	data, err := yaml.Marshal(values)
	if err != nil {
		return err
	}

	secret := &corev1.Secret{}
	secret.Name = fmt.Sprintf("%s-export", h.DeployItem.Name)
	secret.Namespace = h.DeployItem.Namespace
	if h.DeployItem.Status.ExportReference != nil {
		secret.Name = h.DeployItem.Status.ExportReference.Name
		secret.Namespace = h.DeployItem.Status.ExportReference.Namespace
	}

	_, err = controllerutil.CreateOrUpdate(ctx, h.kubeClient, secret, func() error {
		secret.Data = map[string][]byte{
			lsv1alpha1.DataObjectSecretDataKey: data,
		}
		return controllerutil.SetOwnerReference(h.DeployItem, secret, kubernetes.LandscaperScheme)
	})
	if err != nil {
		return err
	}

	h.DeployItem.Status.ExportReference = &lsv1alpha1.ObjectReference{
		Name:      secret.Name,
		Namespace: secret.Namespace,
	}

	return h.kubeClient.Status().Update(ctx, h.DeployItem)
}

// DeleteFiles deletes the managed resources from the target cluster.
func (h *Helm) DeleteFiles(ctx context.Context) error {
	h.log.Info("Deleting files.")
	h.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseDeleting

	if h.ProviderStatus == nil || len(h.ProviderStatus.ManagedResources) == 0 {
		controllerutil.RemoveFinalizer(h.DeployItem, lsv1alpha1.LandscaperFinalizer)
		return h.kubeClient.Update(ctx, h.DeployItem)
	}

	_, targetClient, err := h.TargetClient()
	if err != nil {
		return err
	}

	nonCompletedResources := make([]string, 0)
	for _, ref := range h.ProviderStatus.ManagedResources {
		obj := kutil.ObjectFromTypedObjectReference(&ref)
		if err := targetClient.Delete(ctx, obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		nonCompletedResources = append(nonCompletedResources, fmt.Sprintf("%s/%s(%s)", ref.Namespace, ref.Name, ref.Kind))
	}

	if len(nonCompletedResources) != 0 {
		return fmt.Errorf("waiting for the deletion of %q to be completed", strings.Join(nonCompletedResources, ","))
	}

	// remove finalizer
	controllerutil.RemoveFinalizer(h.DeployItem, lsv1alpha1.LandscaperFinalizer)
	return h.kubeClient.Update(ctx, h.DeployItem)
}

// ApplyObject applies a managed resource to the target cluster.
func (h *Helm) ApplyObject(ctx context.Context, kubeClient client.Client, obj *unstructured.Unstructured) error {
	currOp := "ApplyObjects"
	currObj := unstructured.Unstructured{} // can't use obj.NewEmptyInstance() as this returns a runtime.Unstructured object which doesn't implement client.Object
	currObj.GetObjectKind().SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	key := kutil.ObjectKey(obj.GetName(), obj.GetNamespace())
	if err := kubeClient.Get(ctx, key, &currObj); err != nil {
		if !apierrors.IsNotFound(err) {
			return lsv1alpha1helper.NewWrappedError(err,
				currOp, "GetObject", err.Error())
		}
		if err := kubeClient.Create(ctx, obj); err != nil {
			err = fmt.Errorf("unable to create resource %s: %w", key.String(), err)
			return lsv1alpha1helper.NewWrappedError(err,
				currOp, "CreateObject", err.Error())
		}
		return nil
	}

	// Set the required and immutable fields from the current object.
	// Update fails if these fields are missing
	if err := kutil.SetRequiredNestedFieldsFromObj(&currObj, obj); err != nil {
		return err
	}

	switch h.ProviderConfiguration.UpdateStrategy {
	case helmv1alpha1.UpdateStrategyUpdate:
		if err := kubeClient.Update(ctx, obj); err != nil {
			err = fmt.Errorf("unable to update resource %s: %w", key.String(), err)
			return lsv1alpha1helper.NewWrappedError(err,
				currOp, "ApplyObject", err.Error())
		}
	case helmv1alpha1.UpdateStrategyPatch:
		if err := kubeClient.Patch(ctx, &currObj, client.MergeFrom(obj)); err != nil {
			err = fmt.Errorf("unable to patch resource %s: %w", key.String(), err)
			return lsv1alpha1helper.NewWrappedError(err,
				currOp, "ApplyObject", err.Error())
		}
	default:
		err := fmt.Errorf("%s is not a valid update strategy", h.ProviderConfiguration.UpdateStrategy)
		return lsv1alpha1helper.NewWrappedError(err,
			currOp, "ApplyObject", err.Error())
	}
	return nil
}

// cleanupOrphanedResources removes all managed resources that are not rendered anymore.
func (h *Helm) cleanupOrphanedResources(ctx context.Context, kubeClient client.Client, oldObjects []lsv1alpha1.TypedObjectReference, currentObjects []*unstructured.Unstructured) error {
	//objectList := &unstructured.UnstructuredList{}
	//if err := kubeClient.List(ctx, objectList, client.MatchingLabels{helmv1alpha1.ManagedDeployItemLabel: h.DeployItem.Name}); err != nil {
	//	return fmt.Errorf("unable to list all managed resources: %w", err)
	//}
	var (
		allErrs []error
		wg      sync.WaitGroup
	)
	for _, ref := range oldObjects {
		obj := kutil.ObjectFromTypedObjectReference(&ref)
		if err := kubeClient.Get(ctx, kutil.ObjectKey(ref.Name, ref.Namespace), obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("unable to get object %s %s: %w", obj.GroupVersionKind().String(), obj.GetName(), err)
		}

		if !containsUnstructuredObject(obj, currentObjects) {
			wg.Add(1)
			go func(obj *unstructured.Unstructured) {
				defer wg.Done()
				// Delete object and ensure it is deleted from the cluster.
				timeout, _ := time.ParseDuration(h.ProviderConfiguration.DeleteTimeout)
				err := kutil.DeleteAndWaitForObjectDeleted(ctx, kubeClient, timeout, obj)
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

func (h *Helm) constructExportsFromValues(values map[string]interface{}) (map[string]interface{}, error) {
	exports := make(map[string]interface{})

	for _, export := range h.ProviderConfiguration.ExportsFromManifests {
		if export.FromResource != nil {
			continue
		}

		var val interface{}
		if err := jsonpath.GetValue(export.JSONPath, values, &val); err != nil {
			return nil, err
		}

		newValue, err := jsonpath.Construct(export.Key, val)
		if err != nil {
			return nil, err
		}

		exports = utils.MergeMaps(exports, newValue)
	}

	return exports, nil
}

func (h *Helm) injectLabels(obj *unstructured.Unstructured) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[helmv1alpha1.ManagedDeployItemLabel] = h.DeployItem.Name
	obj.SetLabels(labels)
}

func (h *Helm) addExport(exports map[string]interface{}, obj *unstructured.Unstructured) (map[string]interface{}, error) {
	export := h.findResource(obj)
	if export == nil {
		return exports, nil
	}

	var val interface{}
	if err := jsonpath.GetValue(export.JSONPath, obj.Object, &val); err != nil {
		return nil, err
	}

	newValue, err := jsonpath.Construct(export.Key, val)
	if err != nil {
		return nil, err
	}

	return utils.MergeMaps(exports, newValue), nil
}

func (h *Helm) findResource(obj *unstructured.Unstructured) *helmv1alpha1.ExportFromManifestItem {
	for _, export := range h.ProviderConfiguration.ExportsFromManifests {
		if export.FromResource == nil {
			continue
		}
		if export.FromResource.APIVersion != obj.GetAPIVersion() {
			continue
		}
		if export.FromResource.Kind != obj.GetKind() {
			continue
		}
		if export.FromResource.Name != obj.GetName() {
			continue
		}
		if export.FromResource.Namespace != obj.GetNamespace() {
			continue
		}
		return &export
	}
	return nil
}

func encodeStatus(status *helmv1alpha1.ProviderStatus) (*runtime.RawExtension, error) {
	status.TypeMeta = metav1.TypeMeta{
		APIVersion: helmv1alpha1.SchemeGroupVersion.String(),
		Kind:       "ProviderStatus",
	}

	raw := &runtime.RawExtension{}
	obj := status.DeepCopyObject()
	if err := runtime.Convert_runtime_Object_To_runtime_RawExtension(&obj, raw, nil); err != nil {
		return nil, err
	}
	return raw, nil
}
