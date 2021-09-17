// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/deployer/lib/resourcemanager"

	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"

	lserrors "github.com/gardener/landscaper/apis/errors"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	health "github.com/gardener/landscaper/pkg/deployer/lib/readinesscheck"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"
	"github.com/gardener/landscaper/pkg/utils"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// ApplyFiles applies the helm templated files to the target cluster.
func (h *Helm) ApplyFiles(ctx context.Context, files map[string]string, exports map[string]interface{}) error {
	currOp := "ApplyFile"
	_, targetClient, targetClientSet, err := h.TargetClient(ctx)
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
		Clientset:        targetClientSet,
		DefaultNamespace: h.ProviderConfiguration.Namespace,
		DeployItemName:   h.DeployItem.Name,
		DeleteTimeout:    h.ProviderConfiguration.DeleteTimeout.Duration,
		UpdateStrategy:   manifestv1alpha2.UpdateStrategy(h.ProviderConfiguration.UpdateStrategy),
		Manifests:        manifests,
		ManagedResources: h.ProviderStatus.ManagedResources,
		Labels: map[string]string{
			helmv1alpha1.ManagedDeployItemLabel: h.DeployItem.Name,
		},
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

	exportDefinition := &managedresource.Exports{}
	if h.ProviderConfiguration.Exports != nil {
		exportDefinition = h.ProviderConfiguration.Exports
	}
	if len(h.ProviderConfiguration.ExportsFromManifests) != 0 {
		exportDefinition.Exports = append(exportDefinition.Exports, h.ProviderConfiguration.ExportsFromManifests...)
	}
	if len(exportDefinition.Exports) != 0 {
		opts := resourcemanager.ExporterOptions{
			KubeClient: targetClient,
			Objects:    applier.GetManagedResourcesStatus(),
		}
		if h.Configuration.Export.DefaultTimeout != nil {
			opts.DefaultTimeout = &h.Configuration.Export.DefaultTimeout.Duration
		}
		resourceExports, err := resourcemanager.NewExporter(h.log, opts).
			Export(ctx, exportDefinition)
		if err != nil {
			return lserrors.NewWrappedError(err,
				currOp, "ReadExportValues", err.Error())
		}
		exports = utils.MergeMaps(exports, resourceExports)
	}

	return deployerlib.CreateOrUpdateExport(ctx, h.lsKubeClient, h.DeployItem, exports)
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

// DeleteFiles deletes the managed resources from the target cluster.
func (h *Helm) DeleteFiles(ctx context.Context) error {
	h.log.Info("Deleting files")
	h.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseDeleting

	if h.ProviderStatus == nil || len(h.ProviderStatus.ManagedResources) == 0 {
		controllerutil.RemoveFinalizer(h.DeployItem, lsv1alpha1.LandscaperFinalizer)
		return h.lsKubeClient.Update(ctx, h.DeployItem)
	}

	_, targetClient, _, err := h.TargetClient(ctx)
	if err != nil {
		return err
	}

	nonCompletedResources := make([]string, 0)
	for i := len(h.ProviderStatus.ManagedResources) - 1; i >= 0; i-- {
		ref := h.ProviderStatus.ManagedResources[i]
		obj := kutil.ObjectFromCoreObjectReference(&ref.Resource)
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

func (h *Helm) constructExportsFromValues(values map[string]interface{}) (map[string]interface{}, error) {
	exports := make(map[string]interface{})

	exportDefs := h.ProviderConfiguration.ExportsFromManifests
	if h.ProviderConfiguration.Exports != nil {
		exportDefs = append(exportDefs, h.ProviderConfiguration.Exports.Exports...)
	}
	for _, export := range exportDefs {
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
