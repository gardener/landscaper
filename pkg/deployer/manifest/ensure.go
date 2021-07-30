// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"context"
	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/gardener/landscaper/pkg/deployer/lib/resourcemanager"

	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	lserrors "github.com/gardener/landscaper/apis/errors"
	health "github.com/gardener/landscaper/pkg/deployer/lib/readinesscheck"
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
			ManagedResources: make([]managedresource.ManagedResourceStatus, 0),
		}
	}

	applier := resourcemanager.NewManifestApplier(m.log, resourcemanager.ManifestApplierOptions{
		Decoder:          serializer.NewCodecFactory(Scheme).UniversalDecoder(),
		KubeClient:       targetClient,
		DeployItemName:   m.DeployItem.Name,
		DeleteTimeout:    m.ProviderConfiguration.DeleteTimeout.Duration,
		UpdateStrategy:   m.ProviderConfiguration.UpdateStrategy,
		Manifests:        m.ProviderConfiguration.Manifests,
		ManagedResources: m.ProviderStatus.ManagedResources,
	})

	err = applier.Apply(ctx)
	m.ProviderStatus.ManagedResources = applier.GetManagedResourcesStatus()
	if err != nil {
		var err2 error
		m.DeployItem.Status.ProviderStatus, err2 = kutil.ConvertToRawExtension(m.ProviderStatus, Scheme)
		if err2 != nil {
			m.log.Error(err, "unable to encode status")
		}
		return err
	}

	m.DeployItem.Status.ProviderStatus, err = kutil.ConvertToRawExtension(m.ProviderStatus, Scheme)
	if err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "ProviderStatus", err.Error())
	}
	if err := m.lsKubeClient.Status().Update(ctx, m.DeployItem); err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "UpdateStatus", err.Error())
	}

	return m.CheckResourcesReady(ctx, targetClient)
}

// CheckResourcesReady checks if the managed resources are Ready/Healthy.
func (m *Manifest) CheckResourcesReady(ctx context.Context, client client.Client) error {
	var managedResources []lsv1alpha1.TypedObjectReference
	for _, mr := range m.ProviderStatus.ManagedResources {
		if mr.Policy == managedresource.IgnorePolicy {
			continue
		}
		managedResources = append(managedResources, mr.Resource)
	}

	if !m.ProviderConfiguration.ReadinessChecks.DisableDefault {
		defaultReadinessCheck := health.DefaultReadinessCheck{
			Context:          ctx,
			Client:           client,
			CurrentOp:        "DefaultCheckResourcesReadinessManifest",
			Log:              m.log,
			Timeout:          m.ProviderConfiguration.ReadinessChecks.Timeout,
			ManagedResources: managedResources,
		}
		err := defaultReadinessCheck.CheckResourcesReady()
		if err != nil {
			return err
		}
	}

	if m.ProviderConfiguration.ReadinessChecks.CustomReadinessChecks != nil {
		for _, customReadinessCheckConfig := range m.ProviderConfiguration.ReadinessChecks.CustomReadinessChecks {
			customReadinessCheck := health.CustomReadinessCheck{
				Context:          ctx,
				Client:           client,
				Log:              m.log,
				CurrentOp:        "CustomCheckResourcesReadinessManifest",
				Timeout:          m.ProviderConfiguration.ReadinessChecks.Timeout,
				ManagedResources: managedResources,
				Configuration:    customReadinessCheckConfig,
			}
			err := customReadinessCheck.CheckResourcesReady()
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
		if mr.Policy == managedresource.IgnorePolicy || mr.Policy == managedresource.KeepPolicy {
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
