// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	"github.com/gardener/landscaper/pkg/deployer/lib/resourcemanager"

	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"

	lserrors "github.com/gardener/landscaper/apis/errors"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	health "github.com/gardener/landscaper/pkg/deployer/lib/readinesscheck"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"
	"github.com/gardener/landscaper/pkg/utils"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// ApplyFiles applies the helm templated files to the target cluster.
func (h *Helm) ApplyFiles(ctx context.Context, files map[string]string, exports map[string]interface{}) error {
	currOp := "ApplyFile"
	_, targetClient, err := h.TargetClient(ctx)
	if err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "TargetClusterClient", err.Error())
	}

	if h.ProviderStatus == nil {
		h.ProviderStatus = &helmv1alpha1.ProviderStatus{
			TypeMeta: metav1.TypeMeta{
				APIVersion: helmv1alpha1.SchemeGroupVersion.String(),
				Kind:       "ProviderStatus",
			},
			ManagedResources: make(managedresource.ManagedResourceStatusList, 0),
		}
	}

	objects, err := kutil.ParseFilesToRawExtension(h.log, files)
	if err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "DecodeHelmTemplatedObjects", err.Error())
	}
	// create manifests from objects for the applier
	manifests := make([]managedresource.Manifest, len(objects))
	for i, obj := range objects {
		manifests[i] = managedresource.Manifest{
			Policy:   managedresource.ManagePolicy,
			Manifest: obj,
		}
	}
	applier := resourcemanager.NewManifestApplier(h.log, resourcemanager.ManifestApplierOptions{
		Decoder:          serializer.NewCodecFactory(scheme.Scheme).UniversalDecoder(),
		KubeClient:       targetClient,
		DefaultNamespace: h.ProviderConfiguration.Namespace,
		DeployItemName:   h.DeployItem.Name,
		DeleteTimeout:    h.ProviderConfiguration.DeleteTimeout.Duration,
		UpdateStrategy:   manifestv1alpha2.UpdateStrategy(h.ProviderConfiguration.UpdateStrategy),
		Manifests:        manifests,
		ManagedResources: h.ProviderStatus.ManagedResources,
	})

	err = applier.Apply(ctx)
	h.ProviderStatus.ManagedResources = applier.GetManagedResourcesStatus()
	if err != nil {
		var err2 error
		h.DeployItem.Status.ProviderStatus, err2 = kutil.ConvertToRawExtension(h.ProviderStatus, HelmScheme)
		if err2 != nil {
			h.log.Error(err, "unable to encode status")
		}
		return err
	}

	for _, obj := range applier.GetManagedObjects() {
		exports, err = h.addExport(exports, obj)
		if err != nil {
			return lserrors.NewWrappedError(err,
				currOp, "ReadExportValues", err.Error())
		}
		h.injectLabels(obj)
	}

	h.DeployItem.Status.ProviderStatus, err = kutil.ConvertToRawExtension(h.ProviderStatus, HelmScheme)
	if err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "ProviderStatus", err.Error())
	}
	if err := h.lsKubeClient.Status().Update(ctx, h.DeployItem); err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "UpdateStatus", err.Error())
	}

	err = h.CheckResourcesReady(ctx, targetClient)
	if err != nil {
		return err
	}

	if err := h.createOrUpdateExport(ctx, exports); err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "CreateExport", err.Error())
	}

	return nil
}

// CheckResourcesReady checks if the managed resources are Ready/Healthy.
func (h *Helm) CheckResourcesReady(ctx context.Context, client client.Client) error {

	if !h.ProviderConfiguration.ReadinessChecks.DisableDefault {
		defaultReadinessCheck := health.DefaultReadinessCheck{
			Context:          ctx,
			Client:           client,
			CurrentOp:        "DefaultCheckResourcesReadinessHelm",
			Log:              h.log,
			Timeout:          h.ProviderConfiguration.ReadinessChecks.Timeout,
			ManagedResources: h.ProviderStatus.ManagedResources.TypedObjectReferenceList(),
		}
		err := defaultReadinessCheck.CheckResourcesReady()
		if err != nil {
			return err
		}
	}

	if h.ProviderConfiguration.ReadinessChecks.CustomReadinessChecks != nil {
		for _, customReadinessCheckConfig := range h.ProviderConfiguration.ReadinessChecks.CustomReadinessChecks {
			customReadinessCheck := health.CustomReadinessCheck{
				Context:          ctx,
				Client:           client,
				Log:              h.log,
				CurrentOp:        "CustomCheckResourcesReadinessHelm",
				Timeout:          h.ProviderConfiguration.ReadinessChecks.Timeout,
				ManagedResources: h.ProviderStatus.ManagedResources.TypedObjectReferenceList(),
				Configuration:    customReadinessCheckConfig,
			}
			err := customReadinessCheck.CheckResourcesReady()
			if err != nil {
				return err
			}
		}
	}

	h.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
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

	_, err = controllerutil.CreateOrUpdate(ctx, h.lsKubeClient, secret, func() error {
		secret.Data = map[string][]byte{
			lsv1alpha1.DataObjectSecretDataKey: data,
		}
		return controllerutil.SetOwnerReference(h.DeployItem, secret, api.LandscaperScheme)
	})
	if err != nil {
		return err
	}

	h.DeployItem.Status.ExportReference = &lsv1alpha1.ObjectReference{
		Name:      secret.Name,
		Namespace: secret.Namespace,
	}

	return h.lsKubeClient.Status().Update(ctx, h.DeployItem)
}

// DeleteFiles deletes the managed resources from the target cluster.
func (h *Helm) DeleteFiles(ctx context.Context) error {
	h.log.Info("Deleting files.")
	h.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseDeleting

	if h.ProviderStatus == nil || len(h.ProviderStatus.ManagedResources) == 0 {
		controllerutil.RemoveFinalizer(h.DeployItem, lsv1alpha1.LandscaperFinalizer)
		return h.lsKubeClient.Update(ctx, h.DeployItem)
	}

	_, targetClient, err := h.TargetClient(ctx)
	if err != nil {
		return err
	}

	nonCompletedResources := make([]string, 0)
	for _, ref := range h.ProviderStatus.ManagedResources {
		obj := kutil.ObjectFromTypedObjectReference(&ref.Resource)
		if err := targetClient.Delete(ctx, obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		nonCompletedResources = append(nonCompletedResources, fmt.Sprintf("%s/%s(%s)", ref.Resource.Namespace, ref.Resource.Name, ref.Resource.Kind))
	}

	if len(nonCompletedResources) != 0 {
		return fmt.Errorf("waiting for the deletion of %q to be completed", strings.Join(nonCompletedResources, ","))
	}

	// remove finalizer
	controllerutil.RemoveFinalizer(h.DeployItem, lsv1alpha1.LandscaperFinalizer)
	return h.lsKubeClient.Update(ctx, h.DeployItem)
}

// ApplyObject applies a managed resource to the target cluster.
func (h *Helm) ApplyObject(ctx context.Context, kubeClient client.Client, obj *unstructured.Unstructured) error {
	currOp := "ApplyObjects"
	currObj := unstructured.Unstructured{} // can't use obj.NewEmptyInstance() as this returns a runtime.Unstructured object which doesn't implement client.Object
	currObj.GetObjectKind().SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	key := kutil.ObjectKey(obj.GetName(), obj.GetNamespace())
	if err := kubeClient.Get(ctx, key, &currObj); err != nil {
		if !apierrors.IsNotFound(err) {
			return lserrors.NewWrappedError(err,
				currOp, "GetObject", err.Error())
		}
		if err := kubeClient.Create(ctx, obj); err != nil {
			err = fmt.Errorf("unable to create resource %s: %w", key.String(), err)
			return lserrors.NewWrappedError(err,
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
			return lserrors.NewWrappedError(err,
				currOp, "ApplyObject", err.Error())
		}
	case helmv1alpha1.UpdateStrategyPatch:
		if err := kubeClient.Patch(ctx, obj, client.MergeFrom(&currObj)); err != nil {
			err = fmt.Errorf("unable to patch resource %s: %w", key.String(), err)
			return lserrors.NewWrappedError(err,
				currOp, "ApplyObject", err.Error())
		}
	default:
		err := fmt.Errorf("%s is not a valid update strategy", h.ProviderConfiguration.UpdateStrategy)
		return lserrors.NewWrappedError(err,
			currOp, "ApplyObject", err.Error())
	}
	return nil
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

func (h *Helm) findResource(obj *unstructured.Unstructured) *managedresource.Export {
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
