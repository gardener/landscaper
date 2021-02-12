// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"context"
	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/apis/deployer/manifest"
)

func (m *Manifest) Reconcile(ctx context.Context) error {
	currOp := "ReconcileManifests"
	m.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseProgressing

	_, targetClient, err := m.TargetClient()
	if err != nil {
		m.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(m.DeployItem.Status.LastError,
			currOp, "TargetClusterClient", err.Error())
		return err
	}

	if m.ProviderStatus == nil {
		m.ProviderStatus = &manifest.ProviderStatus{
			TypeMeta: metav1.TypeMeta{
				APIVersion: manifest.SchemeGroupVersion.String(),
				Kind:       "ProviderStatus",
			},
			ManagedResources: make([]manifest.ManagedResourceStatus, 0),
		}
	}

	applier := &ObjectApplier{
		log:              m.log,
		decoder:          serializer.NewCodecFactory(ManifestScheme).UniversalDecoder(),
		kubeClient:       targetClient,
		deployItemName:   m.DeployItem.Name,
		updateStrategy:   m.ProviderConfiguration.UpdateStrategy,
		manifests:        m.ProviderConfiguration.Manifests,
		managedResources: m.ProviderStatus.ManagedResources,
	}

	err = applier.Apply(ctx)
	m.ProviderStatus.ManagedResources = applier.managedResources
	if err != nil {
		var err2 error
		m.DeployItem.Status.ProviderStatus, err2 = encodeStatus(m.ProviderStatus)
		if err2 != nil {
			m.log.Error(err, "unable to encode status")
		}
		return err
	}

	m.DeployItem.Status.ProviderStatus, err = encodeStatus(m.ProviderStatus)
	if err != nil {
		m.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(m.DeployItem.Status.LastError,
			currOp, "ProviderStatus", err.Error())
		return err
	}

	m.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
	m.DeployItem.Status.ObservedGeneration = m.DeployItem.Generation
	m.DeployItem.Status.LastError = nil
	return m.kubeClient.Status().Update(ctx, m.DeployItem)
}

func (m *Manifest) Delete(ctx context.Context) error {
	currOp := "DeleteManifests"
	m.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseDeleting

	if m.ProviderStatus == nil || len(m.ProviderStatus.ManagedResources) == 0 {
		controllerutil.RemoveFinalizer(m.DeployItem, lsv1alpha1.LandscaperFinalizer)
		return m.kubeClient.Update(ctx, m.DeployItem)
	}

	_, kubeClient, err := m.TargetClient()
	if err != nil {
		m.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(m.DeployItem.Status.LastError,
			currOp, "TargetClusterClient", err.Error())
		return err
	}

	completed := true
	for _, mr := range m.ProviderStatus.ManagedResources {
		if mr.Policy == manifest.IgnorePolicy || mr.Policy == manifest.KeepPolicy {
			continue
		}
		ref := mr.Resource
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
		if err := kubeClient.Delete(ctx, obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			m.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(m.DeployItem.Status.LastError,
				currOp, "DeleteManifest", err.Error())
			return err
		}
		completed = false
	}

	if !completed {
		m.DeployItem.Status.LastError = nil
		return errors.New("not all items are deleted")
	}

	// remove finalizer
	controllerutil.RemoveFinalizer(m.DeployItem, lsv1alpha1.LandscaperFinalizer)
	return m.kubeClient.Update(ctx, m.DeployItem)
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

func encodeStatus(status *manifest.ProviderStatus) (*runtime.RawExtension, error) {
	status.TypeMeta = metav1.TypeMeta{
		APIVersion: manifest.SchemeGroupVersion.String(),
		Kind:       "ProviderStatus",
	}

	raw := &runtime.RawExtension{}
	obj := status.DeepCopyObject()
	if err := runtime.Convert_runtime_Object_To_runtime_RawExtension(&obj, raw, nil); err != nil {
		return nil, err
	}
	return raw, nil
}
