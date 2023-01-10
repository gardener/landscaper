// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"context"
	"errors"

	"github.com/imdario/mergo"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	lserrors "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	health "github.com/gardener/landscaper/pkg/deployer/lib/readinesscheck"
	"github.com/gardener/landscaper/pkg/deployer/lib/resourcemanager"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

func (m *Manifest) Reconcile(ctx context.Context) error {
	currOp := "ReconcileManifests"
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, currOp})

	m.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseProgressing

	_, targetClient, targetClientSet, err := m.TargetClient(ctx)
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

	applier := resourcemanager.NewManifestApplier(resourcemanager.ManifestApplierOptions{
		Decoder:          serializer.NewCodecFactory(Scheme).UniversalDecoder(),
		KubeClient:       targetClient,
		Clientset:        targetClientSet,
		DeployItemName:   m.DeployItem.Name,
		DeleteTimeout:    m.ProviderConfiguration.DeleteTimeout.Duration,
		UpdateStrategy:   m.ProviderConfiguration.UpdateStrategy,
		Manifests:        m.ProviderConfiguration.Manifests,
		ManagedResources: m.ProviderStatus.ManagedResources,
		Labels: map[string]string{
			manifestv1alpha2.ManagedDeployItemLabel: m.DeployItem.Name,
		},
	})

	err = applier.Apply(ctx)
	m.ProviderStatus.ManagedResources = applier.GetManagedResourcesStatus()
	if err != nil {
		var err2 error
		m.DeployItem.Status.ProviderStatus, err2 = kutil.ConvertToRawExtension(m.ProviderStatus, Scheme)
		if err2 != nil {
			logger.Error(err, "unable to encode status")
		}
		return err
	}

	m.DeployItem.Status.ProviderStatus, err = kutil.ConvertToRawExtension(m.ProviderStatus, Scheme)
	if err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "ProviderStatus", err.Error())
	}
	if err := m.Writer().UpdateDeployItemStatus(ctx, read_write_layer.W000062, m.DeployItem); err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "UpdateStatus", err.Error())
	}

	if err := m.CheckResourcesReady(ctx, targetClient); err != nil {
		return err
	}

	if m.ProviderConfiguration.Exports != nil {
		opts := resourcemanager.ExporterOptions{
			KubeClient: targetClient,
			Objects:    applier.GetManagedResourcesStatus(),
			DeployItem: m.DeployItem,
			LsClient:   m.lsKubeClient,
		}
		if m.Configuration.Export.DefaultTimeout != nil {
			opts.DefaultTimeout = &m.Configuration.Export.DefaultTimeout.Duration
		}
		exporter := resourcemanager.NewExporter(opts)
		exports, err := exporter.Export(ctx, m.ProviderConfiguration.Exports)
		if err != nil {
			return lserrors.NewWrappedError(err, currOp, "ReadExportValues", err.Error())
		}

		if err := deployerlib.CreateOrUpdateExport(ctx, m.Writer(), m.lsKubeClient, m.DeployItem, exports); err != nil {
			return err
		}
	}

	m.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded

	return nil
}

// CheckResourcesReady checks if the managed resources are Ready/Healthy.
func (m *Manifest) CheckResourcesReady(ctx context.Context, client client.Client) error {

	managedresources := m.ProviderStatus.ManagedResources.TypedObjectReferenceList()
	if !m.ProviderConfiguration.ReadinessChecks.DisableDefault {
		defaultReadinessCheck := health.DefaultReadinessCheck{
			Context:             ctx,
			Client:              client,
			CurrentOp:           "DefaultCheckResourcesReadinessManifest",
			Timeout:             m.ProviderConfiguration.ReadinessChecks.Timeout,
			ManagedResources:    managedresources,
			FailOnMissingObject: true,
		}
		err := defaultReadinessCheck.CheckResourcesReady()
		if err != nil {
			return err
		}
	}

	if m.ProviderConfiguration.ReadinessChecks.CustomReadinessChecks != nil {
		for _, customReadinessCheckConfig := range m.ProviderConfiguration.ReadinessChecks.CustomReadinessChecks {
			timeout := customReadinessCheckConfig.Timeout
			if timeout == nil {
				timeout = m.ProviderConfiguration.ReadinessChecks.Timeout
			}
			customReadinessCheck := health.CustomReadinessCheck{
				Context:          ctx,
				Client:           client,
				CurrentOp:        "CustomCheckResourcesReadinessManifest",
				Timeout:          timeout,
				ManagedResources: managedresources,
				Configuration:    customReadinessCheckConfig,
			}
			err := customReadinessCheck.CheckResourcesReady()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *Manifest) Delete(ctx context.Context) error {
	logger, ctx := logging.FromContextOrNew(ctx, nil, lc.KeyMethod, "Delete")

	currOp := "DeleteManifests"
	m.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseDeleting

	if m.ProviderStatus == nil || len(m.ProviderStatus.ManagedResources) == 0 {
		controllerutil.RemoveFinalizer(m.DeployItem, lsv1alpha1.LandscaperFinalizer)
		return m.Writer().UpdateDeployItem(ctx, read_write_layer.W000044, m.DeployItem)
	}

	_, kubeClient, _, err := m.TargetClient(ctx)
	if err != nil {
		return lserrors.NewWrappedError(err,
			currOp, "TargetClusterClient", err.Error())
	}

	completed := true
	for i := len(m.ProviderStatus.ManagedResources) - 1; i >= 0; i-- {
		mr := m.ProviderStatus.ManagedResources[i]
		if mr.Policy == managedresource.IgnorePolicy || mr.Policy == managedresource.KeepPolicy {
			continue
		}
		ref := mr.Resource
		obj := kutil.ObjectFromCoreObjectReference(&ref)

		currObj := unstructured.Unstructured{} // can't use obj.NewEmptyInstance() as this returns a runtime.Unstructured object which doesn't implement client.Object
		currObj.GetObjectKind().SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
		key := kutil.ObjectKey(obj.GetName(), obj.GetNamespace())
		if err := kubeClient.Get(ctx, key, &currObj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return lserrors.NewWrappedError(err,
				currOp, "GetManifest", err.Error())
		}

		// if fallback policy is set and the resource is already managed by another deployer
		// we are not allowed to delete that resource
		if mr.Policy == managedresource.FallbackPolicy && !kutil.HasLabelWithValue(&currObj, manifestv1alpha2.ManagedDeployItemLabel, m.DeployItem.Name) {
			logger.Info("Resource is already managed, skip delete", lc.KeyResource, key.String())
			continue
		}

		if m.ProviderConfiguration.Manifests[i].AnnotateBeforeDelete != nil {
			objAnnotations := currObj.GetAnnotations()
			if objAnnotations == nil {
				objAnnotations = m.ProviderConfiguration.Manifests[i].AnnotateBeforeDelete
			} else {
				if err := mergo.Merge(&objAnnotations, m.ProviderConfiguration.Manifests[i].AnnotateBeforeDelete, mergo.WithOverride); err != nil {
					logger.Error(err, "unable to merge resource annotations with before delete annotations", lc.KeyResource, key.String())
				}
			}

			currObj.SetAnnotations(objAnnotations)

			if err := kubeClient.Update(ctx, &currObj); err != nil {
				logger.Error(err, "unable to update resource with before delete annotations", lc.KeyResource, key.String())
			}
		}

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
		return errors.New("not all items are deleted")
	}
	controllerutil.RemoveFinalizer(m.DeployItem, lsv1alpha1.LandscaperFinalizer)
	return m.Writer().UpdateDeployItem(ctx, read_write_layer.W000045, m.DeployItem)
}

func (m *Manifest) Writer() *read_write_layer.Writer {
	return read_write_layer.NewWriter(m.lsKubeClient)
}
