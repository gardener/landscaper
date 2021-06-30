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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	"github.com/gardener/landscaper/pkg/deployer/lib/healthcheck"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

func (m *Manifest) Reconcile(ctx context.Context) error {
	currOp := "ReconcileManifests"
	m.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseProgressing

	_, targetClient, err := m.TargetClient(ctx)
	if err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "TargetClusterClient", err.Error())
	}

	if m.ProviderStatus == nil {
		m.ProviderStatus = &manifestv1alpha2.ProviderStatus{
			TypeMeta: metav1.TypeMeta{
				APIVersion: manifestv1alpha2.SchemeGroupVersion.String(),
				Kind:       "ProviderStatus",
			},
			ManagedResources: make([]manifestv1alpha2.ManagedResourceStatus, 0),
		}
	}

	applier := &ObjectApplier{
		log:              m.log,
		decoder:          serializer.NewCodecFactory(Scheme).UniversalDecoder(),
		kubeClient:       targetClient,
		deployItemName:   m.DeployItem.Name,
		deleteTimeout:    *m.ProviderConfiguration.DeleteTimeout,
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
		return lserrors.NewWrappedError(err,
			currOp, "ProviderStatus", err.Error())
	}
	if err := m.lsKubeClient.Status().Update(ctx, m.DeployItem); err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "UpdateStatus", err.Error())
	}

	return m.CheckResourcesHealth(ctx, targetClient)
}

// CheckResourcesHealth checks if the managed resources are Ready/Healthy.
func (m *Manifest) CheckResourcesHealth(ctx context.Context, client client.Client) error {
	var managedResources []lsv1alpha1.TypedObjectReference
	for _, mr := range m.ProviderStatus.ManagedResources {
		if mr.Policy == manifestv1alpha2.IgnorePolicy {
			continue
		}
		managedResources = append(managedResources, mr.Resource)
	}

	if !m.ProviderConfiguration.HealthChecks.DisableDefault {
		defaultHealthCheck := healthcheck.DefaultHealthCheck{
			Context:          ctx,
			Client:           client,
			CurrentOp:        "DefaultCheckResourcesHealthManifest",
			Log:              m.log,
			Timeout:          m.ProviderConfiguration.HealthChecks.Timeout,
			ManagedResources: managedResources,
		}
		err := defaultHealthCheck.CheckResourcesHealth()
		if err != nil {
			return err
		}
	}

	if m.ProviderConfiguration.HealthChecks.CustomHealthChecks != nil {
		for _, customHealthCheckConfig := range m.ProviderConfiguration.HealthChecks.CustomHealthChecks {
			customHealthCheck := healthcheck.CustomHealthCheck{
				Context:          ctx,
				Client:           client,
				Log:              m.log,
				CurrentOp:        "CustomCheckResourcesHealthManifest",
				Timeout:          m.ProviderConfiguration.HealthChecks.Timeout,
				ManagedResources: managedResources,
				Configuration:    customHealthCheckConfig,
			}
			err := customHealthCheck.CheckResourcesHealth()
			if err != nil {
				return err
			}
		}
	}

	m.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
	m.DeployItem.Status.LastError = nil
	return nil
}

func (m *Manifest) Delete(ctx context.Context) error {
	currOp := "DeleteManifests"
	m.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseDeleting

	if m.ProviderStatus == nil || len(m.ProviderStatus.ManagedResources) == 0 {
		controllerutil.RemoveFinalizer(m.DeployItem, lsv1alpha1.LandscaperFinalizer)
		return m.lsKubeClient.Update(ctx, m.DeployItem)
	}

	_, kubeClient, err := m.TargetClient(ctx)
	if err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "TargetClusterClient", err.Error())
	}

	completed := true
	for _, mr := range m.ProviderStatus.ManagedResources {
		if mr.Policy == manifestv1alpha2.IgnorePolicy || mr.Policy == manifestv1alpha2.KeepPolicy {
			continue
		}
		ref := mr.Resource
		obj := kutil.ObjectFromTypedObjectReference(&ref)
		if err := kubeClient.Delete(ctx, obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return lserrors.NewWrappedError(err,
				currOp, "DeleteManifest", err.Error())
		}
		completed = false
	}

	if !completed {
		m.DeployItem.Status.LastError = nil
		return errors.New("not all items are deleted")
	}

	// remove finalizer
	controllerutil.RemoveFinalizer(m.DeployItem, lsv1alpha1.LandscaperFinalizer)
	return m.lsKubeClient.Update(ctx, m.DeployItem)
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

func encodeStatus(status *manifestv1alpha2.ProviderStatus) (*runtime.RawExtension, error) {
	status.TypeMeta = metav1.TypeMeta{
		APIVersion: manifestv1alpha2.SchemeGroupVersion.String(),
		Kind:       "ProviderStatus",
	}

	raw := &runtime.RawExtension{}
	obj := status.DeepCopyObject()
	if err := runtime.Convert_runtime_Object_To_runtime_RawExtension(&obj, raw, nil); err != nil {
		return nil, err
	}
	return raw, nil
}
