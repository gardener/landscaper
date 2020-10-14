// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	apimacherrors "k8s.io/apimachinery/pkg/util/errors"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	helmv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"
	"github.com/gardener/landscaper/pkg/utils"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

func (h *Helm) ApplyFiles(ctx context.Context, files map[string]string, exports map[string]interface{}) error {
	currOp := "ApplyFile"
	_, targetClient, err := h.TargetClient()
	if err != nil {
		h.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(h.DeployItem.Status.LastError,
			currOp, "TargetClusterClient", err.Error())
		return err
	}

	objects := make([]*unstructured.Unstructured, 0)
	for name, content := range files {
		decodedObjects, err := h.decodeObjects(name, []byte(content))
		if err != nil {
			h.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(h.DeployItem.Status.LastError,
				currOp, "DecodeHelmTemplatedObjects", err.Error())
			return err
		}
		// add possible export
		for _, obj := range decodedObjects {
			exports, err = h.addExport(exports, obj)
			if err != nil {
				h.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(h.DeployItem.Status.LastError,
					currOp, "ReadExportValues", err.Error())
				return err
			}
			h.injectLabels(obj)
		}

		objects = append(objects, decodedObjects...)
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

	if h.ProviderStatus != nil {
		if err := h.cleanupOrphanedResources(ctx, targetClient, h.ProviderStatus.ManagedResources, objects); err != nil {
			h.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(h.DeployItem.Status.LastError,
				currOp, "CleanupOrphanedResources", err.Error())
			return fmt.Errorf("unable to cleanup orphaned resources: %w", err)
		}
	}

	status := &helmv1alpha1.ProviderStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: helmv1alpha1.SchemeGroupVersion.String(),
			Kind:       "ProviderStatus",
		},
		ManagedResources: managedResources,
	}
	statusData, err := encodeStatus(status)
	if err != nil {
		h.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(h.DeployItem.Status.LastError,
			currOp, "ProviderStatus", err.Error())
		return err
	}

	if err := h.createOrUpdateExport(ctx, exports); err != nil {
		h.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(h.DeployItem.Status.LastError,
			currOp, "CreateExport", err.Error())
		return err
	}

	h.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
	h.DeployItem.Status.ProviderStatus = statusData
	h.DeployItem.Status.ObservedGeneration = h.DeployItem.Generation
	if err := h.kubeClient.Status().Update(ctx, h.DeployItem); err != nil {
		h.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(h.DeployItem.Status.LastError,
			currOp, "UpdateStatus", err.Error())
		return err
	}
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

func (h *Helm) DeleteFiles(ctx context.Context) error {
	h.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseDeleting
	status := &helmv1alpha1.ProviderStatus{}
	if _, _, err := serializer.NewCodecFactory(Helmscheme).UniversalDecoder().Decode(h.DeployItem.Status.ProviderStatus.Raw, nil, status); err != nil {
		return err
	}

	if len(status.ManagedResources) == 0 {
		controllerutil.RemoveFinalizer(&h.DeployItem.ObjectMeta, lsv1alpha1.LandscaperFinalizer)
		return h.kubeClient.Update(ctx, h.DeployItem)
	}

	_, targetClient, err := h.TargetClient()
	if err != nil {
		return err
	}

	completed := true
	for _, ref := range status.ManagedResources {
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": ref.APIVersion,
				"kind":       ref.Kind,
				"metadata": map[string]interface{}{
					"name":      ref.Name,
					"namespace": ref.Namespace,
				},
			},
		}
		if err := targetClient.Delete(ctx, obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		completed = false
	}

	if !completed {
		return errors.New("not all items are deleted")
	}

	// remove finalizer
	controllerutil.RemoveFinalizer(&h.DeployItem.ObjectMeta, lsv1alpha1.LandscaperFinalizer)
	return h.kubeClient.Update(ctx, h.DeployItem)
}

// ApplyObject applies a managed resource to the target cluster.
func (h *Helm) ApplyObject(ctx context.Context, kubeClient client.Client, obj *unstructured.Unstructured) error {
	currOp := "ApplyObjects"
	currObj := obj.NewEmptyInstance()
	key := kutil.ObjectKey(obj.GetName(), obj.GetNamespace())
	if err := kubeClient.Get(ctx, key, currObj); err != nil {
		if !apierrors.IsNotFound(err) {
			h.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(h.DeployItem.Status.LastError,
				currOp, "GetObject", err.Error())
			return err
		}
		if err := kubeClient.Create(ctx, obj); err != nil {
			err = fmt.Errorf("unable to create resource %s: %w", key.String(), err)
			h.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(h.DeployItem.Status.LastError,
				currOp, "CreateObject", err.Error())
			return err
		}
		return nil
	}

	switch h.ProviderConfiguration.UpdateStrategy {
	case helmv1alpha1.UpdateStrategyUpdate:
		if err := kubeClient.Update(ctx, obj); err != nil {
			err = fmt.Errorf("unable to update resource %s: %w", key.String(), err)
			h.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(h.DeployItem.Status.LastError,
				currOp, "ApplyObject", err.Error())
			return err
		}
	case helmv1alpha1.UpdateStrategyPatch:
		if err := kubeClient.Patch(ctx, currObj, client.MergeFrom(obj)); err != nil {
			err = fmt.Errorf("unable to patch resource %s: %w", key.String(), err)
			h.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(h.DeployItem.Status.LastError,
				currOp, "ApplyObject", err.Error())
			return err
		}
	default:
		err := fmt.Errorf("%s is not a valid update strategy", h.ProviderConfiguration.UpdateStrategy)
		h.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(h.DeployItem.Status.LastError,
			currOp, "ApplyObject", err.Error())
		return err
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
		uObj := unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": ref.APIVersion,
				"kind":       ref.Kind,
				"metadata": map[string]interface{}{
					"name":      ref.Name,
					"namespace": ref.Namespace,
				},
			},
		}
		if err := kubeClient.Get(ctx, kutil.ObjectKey(ref.Name, ref.Namespace), &uObj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("unable to get object %s %s: %w", uObj.GroupVersionKind().String(), uObj.GetName(), err)
		}

		if !containsUnstructuredObject(&uObj, currentObjects) {
			wg.Add(1)
			go func(obj unstructured.Unstructured) {
				defer wg.Done()
				if err := kubeClient.Delete(ctx, &obj); err != nil {
					allErrs = append(allErrs, fmt.Errorf("unable to delete %s %s/%s: %w", obj.GroupVersionKind().String(), obj.GetName(), obj.GetNamespace(), err))
				}
				// todo: wait for deletion
			}(uObj)
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

func (h *Helm) decodeObjects(name string, data []byte) ([]*unstructured.Unstructured, error) {
	var (
		decoder    = yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(data), 1024)
		decodedObj map[string]interface{}
		objects    = make([]*unstructured.Unstructured, 0)
	)

	for i := 0; true; i++ {
		if err := decoder.Decode(&decodedObj); err != nil {
			if err == io.EOF {
				break
			}
			h.log.Error(err, fmt.Sprintf("unable to decode resource %d of file %s", i, name))
			continue
		}

		if decodedObj == nil {
			continue
		}
		obj := &unstructured.Unstructured{Object: decodedObj}
		objects = append(objects, obj.DeepCopy())
	}
	return objects, nil
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
