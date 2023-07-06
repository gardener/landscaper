// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	lserrors "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/deployer/helm/realhelmdeployer"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	health "github.com/gardener/landscaper/pkg/deployer/lib/readinesscheck"
	"github.com/gardener/landscaper/pkg/deployer/lib/resourcemanager"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"
	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

// ApplyFiles applies the helm templated files to the target cluster.
func (h *Helm) ApplyFiles(ctx context.Context, files, crds map[string]string, exports map[string]interface{},
	ch *chart.Chart) error {

	currOp := "ApplyFile"
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, currOp})

	_, targetClient, targetClientSet, err := h.TargetClient(ctx)
	if err != nil {
		return lserrors.NewWrappedError(err, currOp, "TargetClusterClient", err.Error())
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

	var (
		managedResourceStatusList managedresource.ManagedResourceStatusList
		deployErr                 error
	)

	shouldUseRealHelmDeployer := pointer.BoolDeref(h.ProviderConfiguration.HelmDeployment, true)

	if shouldUseRealHelmDeployer {
		// apply helm install/upgrade, and afterwards get the list of deployed resources by helm get release
		realHelmDeployer := realhelmdeployer.NewRealHelmDeployer(ch, h.ProviderConfiguration, h.TargetRestConfig, targetClientSet)
		deployErr = realHelmDeployer.Deploy(ctx)
		if deployErr == nil {
			managedResourceStatusList, err = realHelmDeployer.GetManagedResourcesStatus(ctx)
			if err != nil {
				return err
			}
			h.ProviderStatus.ManagedResources = managedResourceStatusList
		}

	} else {
		manifests, err := h.createManifests(ctx, currOp, files, crds)
		if err != nil {
			return err
		}

		var applier *resourcemanager.ManifestApplier
		applier, deployErr = h.applyManifests(ctx, targetClient, targetClientSet, manifests)
		managedResourceStatusList = applier.GetManagedResourcesStatus()
	}

	// common error handling for deploy errors (h.applyManifests / realHelmDeployer.Deploy)
	if deployErr != nil {
		var err error
		h.DeployItem.Status.ProviderStatus, err = kutil.ConvertToRawExtension(h.ProviderStatus, HelmScheme)
		if err != nil {
			logger.Error(err, "unable to encode status")
		}
		return deployErr
	}

	h.DeployItem.Status.ProviderStatus, err = kutil.ConvertToRawExtension(h.ProviderStatus, HelmScheme)
	if err != nil {
		return lserrors.NewWrappedError(err, currOp, "ProviderStatus", err.Error())
	}

	if err := h.Writer().UpdateDeployItemStatus(ctx, read_write_layer.W000052, h.DeployItem); err != nil {
		return lserrors.NewWrappedError(err, currOp, "UpdateStatus", err.Error())
	}

	if err := h.checkResourcesReady(ctx, targetClient, !shouldUseRealHelmDeployer); err != nil {
		return err
	}

	if err := h.readExportValues(ctx, currOp, targetClient, managedResourceStatusList, exports); err != nil {
		return err
	}

	h.DeployItem.Status.Phase = lsv1alpha1.DeployItemPhases.Succeeded

	return nil
}

func (h *Helm) applyManifests(ctx context.Context, targetClient client.Client, targetClientSet kubernetes.Interface,
	manifests []managedresource.Manifest) (*resourcemanager.ManifestApplier, error) {
	applier := resourcemanager.NewManifestApplier(resourcemanager.ManifestApplierOptions{
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

	err := applier.Apply(ctx)
	h.ProviderStatus.ManagedResources = applier.GetManagedResourcesStatus()

	return applier, err
}

func (h *Helm) createManifests(ctx context.Context, currOp string, files, crds map[string]string) ([]managedresource.Manifest, error) {
	logger, _ := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "createManifests"})

	objects, err := kutil.ParseFilesToRawExtension(logger, files)
	if err != nil {
		return nil, lserrors.NewWrappedError(err,
			currOp, "DecodeHelmTemplatedObjects", err.Error())
	}

	objects, err = deployerlib.ExpandManifests(objects)
	if err != nil {
		return nil, lserrors.NewWrappedError(err, currOp, "ExpandManifests", err.Error())
	}

	crdObjects, err := kutil.ParseFilesToRawExtension(logger, crds)
	if err != nil {
		return nil, lserrors.NewWrappedError(err,
			currOp, "DecodeHelmTemplatedObjects", err.Error())
	}

	// create manifests from objects for the applier
	crdCount := len(crdObjects)
	manifests := make([]managedresource.Manifest, len(objects)+crdCount)
	for i, obj := range crdObjects {
		manifests[i] = managedresource.Manifest{
			Policy:   managedresource.ManagePolicy,
			Manifest: obj,
		}
	}
	for i, obj := range objects {
		manifests[i+crdCount] = managedresource.Manifest{
			Policy:   managedresource.ManagePolicy,
			Manifest: obj,
		}
	}

	if h.ProviderConfiguration.CreateNamespace && len(h.ProviderConfiguration.Namespace) != 0 {
		// add the release namespace as managed resource
		ns := &corev1.Namespace{}
		ns.Name = h.ProviderConfiguration.Namespace
		rawNs, err := kutil.ConvertToRawExtension(ns, scheme.Scheme)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal release namespace: %w", err)
		}
		nsManifest := managedresource.Manifest{
			Policy:   managedresource.KeepPolicy,
			Manifest: rawNs,
		}
		// the namespace is ordered by the manifest deployer, so it is automatically created as first resource
		manifests = append(manifests, nsManifest)
	}

	return manifests, nil
}

// checkResourcesReady checks if the managed resources are Ready/Healthy.
func (h *Helm) checkResourcesReady(ctx context.Context, client client.Client, failOnMissingObject bool) error {

	if !h.ProviderConfiguration.ReadinessChecks.DisableDefault {
		defaultReadinessCheck := health.DefaultReadinessCheck{
			Context:             ctx,
			Client:              client,
			CurrentOp:           "DefaultCheckResourcesReadinessHelm",
			Timeout:             h.ProviderConfiguration.ReadinessChecks.Timeout,
			ManagedResources:    h.ProviderStatus.ManagedResources.TypedObjectReferenceList(),
			FailOnMissingObject: failOnMissingObject,
			InterruptionChecker: deployerlib.NewInterruptionChecker(h.DeployItem, h.lsKubeClient),
		}
		err := defaultReadinessCheck.CheckResourcesReady()
		if err != nil {
			return err
		}
	}

	if h.ProviderConfiguration.ReadinessChecks.CustomReadinessChecks != nil {
		for _, customReadinessCheckConfig := range h.ProviderConfiguration.ReadinessChecks.CustomReadinessChecks {
			timeout := customReadinessCheckConfig.Timeout
			if timeout == nil {
				timeout = h.ProviderConfiguration.ReadinessChecks.Timeout
			}
			customReadinessCheck := health.CustomReadinessCheck{
				Context:             ctx,
				Client:              client,
				CurrentOp:           "CustomCheckResourcesReadinessHelm",
				Timeout:             timeout,
				ManagedResources:    h.ProviderStatus.ManagedResources.TypedObjectReferenceList(),
				Configuration:       customReadinessCheckConfig,
				InterruptionChecker: deployerlib.NewInterruptionChecker(h.DeployItem, h.lsKubeClient),
			}
			err := customReadinessCheck.CheckResourcesReady()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (h *Helm) readExportValues(ctx context.Context, currOp string, targetClient client.Client,
	managedResourceStatusList managedresource.ManagedResourceStatusList, exports map[string]interface{}) error {

	exportDefinition := &managedresource.Exports{}
	if h.ProviderConfiguration.Exports != nil {
		exportDefinition = h.ProviderConfiguration.Exports
	}
	if len(h.ProviderConfiguration.ExportsFromManifests) != 0 {
		exportDefinition.Exports = append(exportDefinition.Exports, h.ProviderConfiguration.ExportsFromManifests...)
	}
	if len(exportDefinition.Exports) != 0 {
		opts := resourcemanager.ExporterOptions{
			KubeClient:          targetClient,
			Objects:             managedResourceStatusList,
			InterruptionChecker: deployerlib.NewInterruptionChecker(h.DeployItem, h.lsKubeClient),
		}
		if h.Configuration.Export.DefaultTimeout != nil {
			opts.DefaultTimeout = &h.Configuration.Export.DefaultTimeout.Duration
		}
		resourceExports, err := resourcemanager.NewExporter(opts).
			Export(ctx, exportDefinition)
		if err != nil {
			return lserrors.NewWrappedError(err,
				currOp, "ReadExportValues", err.Error())
		}
		exports = utils.MergeMaps(exports, resourceExports)
	}

	if err := deployerlib.CreateOrUpdateExport(ctx, h.Writer(), h.lsKubeClient, h.DeployItem, exports); err != nil {
		return err
	}

	return nil
}

// DeleteFiles deletes the managed resources from the target cluster.
func (h *Helm) DeleteFiles(ctx context.Context) error {
	if h.ProviderConfiguration.HelmDeployment != nil && !(*h.ProviderConfiguration.HelmDeployment) {
		return h.deleteManifests(ctx)
	} else {
		return h.deleteManifestsWithRealHelmDeployer(ctx)
	}
}

func (h *Helm) deleteManifests(ctx context.Context) error {
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "deleteManifests"})
	logger.Info("Deleting files")

	h.DeployItem.Status.Phase = lsv1alpha1.DeployItemPhases.Deleting

	if h.ProviderStatus == nil || len(h.ProviderStatus.ManagedResources) == 0 {
		controllerutil.RemoveFinalizer(h.DeployItem, lsv1alpha1.LandscaperFinalizer)
		return h.Writer().UpdateDeployItem(ctx, read_write_layer.W000067, h.DeployItem)
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
			if apierrors.IsNotFound(err) || apimeta.IsNoMatchError(err) {
				// This handles two cases:
				// 1. the resource is already deleted
				// 2. the resource is a custom resource and its CRD is already deleted (and the resourse itself thus too)
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
	return h.Writer().UpdateDeployItem(ctx, read_write_layer.W000049, h.DeployItem)
}

func (h *Helm) deleteManifestsWithRealHelmDeployer(ctx context.Context) error {
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "lsHealthCheckController.check"})
	logger.Info("Deleting files with real helm deployer")

	h.DeployItem.Status.Phase = lsv1alpha1.DeployItemPhases.Deleting

	if h.ProviderStatus == nil {
		controllerutil.RemoveFinalizer(h.DeployItem, lsv1alpha1.LandscaperFinalizer)
		return h.Writer().UpdateDeployItem(ctx, read_write_layer.W000047, h.DeployItem)
	}

	_, _, targetClientSet, err := h.TargetClient(ctx)
	if err != nil {
		return err
	}

	realHelmDeployer := realhelmdeployer.NewRealHelmDeployer(nil, h.ProviderConfiguration,
		h.TargetRestConfig, targetClientSet)

	err = realHelmDeployer.Undeploy(ctx)
	if err == nil {
		return err
	}

	// remove finalizer
	controllerutil.RemoveFinalizer(h.DeployItem, lsv1alpha1.LandscaperFinalizer)
	return h.Writer().UpdateDeployItem(ctx, read_write_layer.W000048, h.DeployItem)
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

func (h *Helm) Writer() *read_write_layer.Writer {
	return read_write_layer.NewWriter(h.lsKubeClient)
}
