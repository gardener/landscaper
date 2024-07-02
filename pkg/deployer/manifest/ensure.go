// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
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
	"github.com/gardener/landscaper/pkg/deployer/lib/interruption"
	health "github.com/gardener/landscaper/pkg/deployer/lib/readinesscheck"
	"github.com/gardener/landscaper/pkg/deployer/lib/resourcemanager"
	"github.com/gardener/landscaper/pkg/deployer/lib/timeout"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

func (m *Manifest) Reconcile(ctx context.Context) error {
	currOp := "ReconcileManifests"
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, currOp})

	if _, err := timeout.TimeoutExceeded(ctx, m.DeployItem, TimeoutCheckpointManifestStartReconcile); err != nil {
		return err
	}

	m.DeployItem.Status.Phase = lsv1alpha1.DeployItemPhases.Progressing

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
		DeployItem:       m.DeployItem,
		UpdateStrategy:   m.ProviderConfiguration.UpdateStrategy,
		Manifests:        m.ProviderConfiguration.Manifests,
		ManagedResources: m.ProviderStatus.ManagedResources,
		Labels: map[string]string{
			manifestv1alpha2.ManagedDeployItemLabel: m.DeployItem.Name,
		},
		DeletionGroupsDuringUpdate: m.ProviderConfiguration.DeletionGroupsDuringUpdate,
		InterruptionChecker:        interruption.NewStandardInterruptionChecker(m.DeployItem, m.lsUncachedClient),
		LsUncachedClient:           m.lsUncachedClient,
	})

	patchInfos, err := applier.Apply(ctx)
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

	if _, err := timeout.TimeoutExceeded(ctx, m.DeployItem, TimeoutCheckpointManifestBeforeReadinessCheck); err != nil {
		return err
	}

	if err := m.CheckResourcesReady(ctx, targetClient); err != nil {
		return err
	}

	if m.ProviderConfiguration.Exports != nil {
		if _, err := timeout.TimeoutExceeded(ctx, m.DeployItem, TimeoutCheckpointManifestBeforeReadingExportValues); err != nil {
			return err
		}

		opts := resourcemanager.ExporterOptions{
			KubeClient:          targetClient,
			InterruptionChecker: interruption.NewStandardInterruptionChecker(m.DeployItem, m.lsUncachedClient),
			LsClient:            m.lsUncachedClient,
			DeployItem:          m.DeployItem,
		}

		exporter := resourcemanager.NewExporter(opts)
		exports, err := exporter.Export(ctx, m.ProviderConfiguration.Exports)
		if err != nil {
			return lserrors.NewWrappedError(err, currOp, "ReadExportValues", err.Error())
		}

		if err := deployerlib.CreateOrUpdateExport(ctx, m.Writer(), m.lsUncachedClient, m.DeployItem, exports); err != nil {
			return err
		}
	}

	err = applier.PatchAfterDeployment(ctx, patchInfos)
	if err != nil {
		return lserrors.NewWrappedError(err, currOp, "PatchAfterDeployment", err.Error())
	}

	m.DeployItem.Status.Phase = lsv1alpha1.DeployItemPhases.Succeeded

	return nil
}

// CheckResourcesReady checks if the managed resources are Ready/Healthy.
func (m *Manifest) CheckResourcesReady(ctx context.Context, client client.Client) error {

	managedresources := m.ProviderStatus.ManagedResources.TypedObjectReferenceList()
	if !m.ProviderConfiguration.ReadinessChecks.DisableDefault {
		timeout, lserr := timeout.TimeoutExceeded(ctx, m.DeployItem, TimeoutCheckpointManifestDefaultReadinessChecks)
		if lserr != nil {
			return lserr
		}

		defaultReadinessCheck := health.DefaultReadinessCheck{
			Context:             ctx,
			Client:              client,
			CurrentOp:           "DefaultCheckResourcesReadinessManifest",
			Timeout:             &lsv1alpha1.Duration{Duration: timeout},
			ManagedResources:    managedresources,
			FailOnMissingObject: true,
			InterruptionChecker: interruption.NewStandardInterruptionChecker(m.DeployItem, m.lsUncachedClient),
		}
		err := defaultReadinessCheck.CheckResourcesReady()
		if err != nil {
			return err
		}
	}

	if m.ProviderConfiguration.ReadinessChecks.CustomReadinessChecks != nil {
		for _, customReadinessCheckConfig := range m.ProviderConfiguration.ReadinessChecks.CustomReadinessChecks {
			timeout, lserr := timeout.TimeoutExceeded(ctx, m.DeployItem, TimeoutCheckpointManifestCustomReadinessChecks)
			if lserr != nil {
				return lserr
			}

			customReadinessCheck := health.CustomReadinessCheck{
				Client:              client,
				CurrentOp:           "CustomCheckResourcesReadinessManifest",
				Timeout:             &lsv1alpha1.Duration{Duration: timeout},
				ManagedResources:    managedresources,
				Configuration:       customReadinessCheckConfig,
				InterruptionChecker: interruption.NewStandardInterruptionChecker(m.DeployItem, m.lsUncachedClient),
				LsClient:            m.lsUncachedClient,
				DeployItem:          m.DeployItem,
			}
			err := customReadinessCheck.CheckResourcesReady(ctx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *Manifest) Delete(ctx context.Context) error {
	return m.deleteManifestsInGroups(ctx)
}

func (m *Manifest) deleteManifestsInGroups(ctx context.Context) error {
	logger, ctx := logging.FromContextOrNew(ctx, nil, lc.KeyMethod, "Delete")
	op := "deleteManifestsInGroups"

	m.DeployItem.Status.Phase = lsv1alpha1.DeployItemPhases.Deleting

	if m.ProviderStatus == nil || len(m.ProviderStatus.ManagedResources) == 0 {
		controllerutil.RemoveFinalizer(m.DeployItem, lsv1alpha1.LandscaperFinalizer)
		return m.Writer().UpdateDeployItem(ctx, read_write_layer.W000044, m.DeployItem)
	}

	if _, err := timeout.TimeoutExceeded(ctx, m.DeployItem, TimeoutCheckpointManifestStartDelete); err != nil {
		return err
	}

	_, targetClient, _, err := m.TargetClient(ctx)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "TargetClusterClient", err.Error())
	}

	managedResources := []managedresource.ManagedResourceStatus{}
	for i := range m.ProviderStatus.ManagedResources {
		mr := &m.ProviderStatus.ManagedResources[i]

		mrLogger, mrCtx := logger.WithValuesAndContext(ctx,
			lc.KeyResource, types.NamespacedName{Namespace: mr.Resource.Namespace, Name: mr.Resource.Name}.String(),
			lc.KeyResourceKind, mr.Resource.Kind)
		mrLogger.Debug("Checking resource")

		ok, err := resourcemanager.FilterByPolicy(mrCtx, mr, targetClient, m.DeployItem.Name)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}

		notFound, err := resourcemanager.AnnotateAndPatchBeforeDelete(ctx, mr, targetClient)
		if err != nil {
			return err
		}
		if notFound {
			continue
		}

		mrLogger.Debug("Object will be deleted")
		managedResources = append(managedResources, *mr)
	}

	interruptionChecker := interruption.NewStandardInterruptionChecker(m.DeployItem, m.lsUncachedClient)

	err = resourcemanager.DeleteManagedResources(
		ctx,
		m.lsUncachedClient,
		managedResources,
		m.ProviderConfiguration.DeletionGroups,
		targetClient,
		m.DeployItem,
		interruptionChecker,
	)
	if err != nil {
		return fmt.Errorf("failed deleting managed resources: %w", err)
	}

	// remove finalizer
	controllerutil.RemoveFinalizer(m.DeployItem, lsv1alpha1.LandscaperFinalizer)
	return m.Writer().UpdateDeployItem(ctx, read_write_layer.W000045, m.DeployItem)
}

func (m *Manifest) Writer() *read_write_layer.Writer {
	return read_write_layer.NewWriter(m.lsUncachedClient)
}
