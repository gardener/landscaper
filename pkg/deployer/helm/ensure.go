// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"fmt"

	"helm.sh/helm/v3/pkg/chart"
	corev1 "k8s.io/api/core/v1"
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
	"github.com/gardener/landscaper/pkg/deployer/lib/interruption"
	health "github.com/gardener/landscaper/pkg/deployer/lib/readinesscheck"
	"github.com/gardener/landscaper/pkg/deployer/lib/resourcemanager"
	"github.com/gardener/landscaper/pkg/deployer/lib/timeout"
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

	var deployErr error

	shouldUseRealHelmDeployer := pointer.BoolDeref(h.ProviderConfiguration.HelmDeployment, true)

	if shouldUseRealHelmDeployer {
		// Apply helm install/upgrade. Afterwards get the list of deployed resources by helm get release.
		// The list is filtered, i.e. it contains only the resources that are needed for the default readiness check.
		realHelmDeployer := realhelmdeployer.NewRealHelmDeployer(ch, h.ProviderConfiguration, h.TargetRestConfig, targetClientSet, h.DeployItem)
		deployErr = realHelmDeployer.Deploy(ctx)
		if deployErr == nil {
			managedResourceStatusList, err := realHelmDeployer.GetManagedResourcesStatus(ctx)
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

		deployErr = h.applyManifests(ctx, targetClient, targetClientSet, manifests)
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

	if _, err := timeout.TimeoutExceeded(ctx, h.DeployItem, TimeoutCheckpointHelmBeforeReadinessCheck); err != nil {
		return err
	}

	if err := h.checkResourcesReady(ctx, targetClient, !shouldUseRealHelmDeployer); err != nil {
		return err
	}

	if _, err := timeout.TimeoutExceeded(ctx, h.DeployItem, TimeoutCheckpointHelmBeforeReadingExportValues); err != nil {
		return err
	}

	if err := h.readExportValues(ctx, currOp, targetClient, exports); err != nil {
		return err
	}

	h.DeployItem.Status.Phase = lsv1alpha1.DeployItemPhases.Succeeded

	return nil
}

func (h *Helm) applyManifests(ctx context.Context, targetClient client.Client, targetClientSet kubernetes.Interface,
	manifests []managedresource.Manifest) error {

	if _, err := timeout.TimeoutExceeded(ctx, h.DeployItem, TimeoutCheckpointHelmStartApplyManifests); err != nil {
		return err
	}

	applier := resourcemanager.NewManifestApplier(resourcemanager.ManifestApplierOptions{
		Decoder:          serializer.NewCodecFactory(scheme.Scheme).UniversalDecoder(),
		KubeClient:       targetClient,
		Clientset:        targetClientSet,
		DefaultNamespace: h.ProviderConfiguration.Namespace,
		DeployItemName:   h.DeployItem.Name,
		DeployItem:       h.DeployItem,
		UpdateStrategy:   manifestv1alpha2.UpdateStrategy(h.ProviderConfiguration.UpdateStrategy),
		Manifests:        manifests,
		ManagedResources: h.ProviderStatus.ManagedResources,
		Labels: map[string]string{
			helmv1alpha1.ManagedDeployItemLabel: h.DeployItem.Name,
		},
		DeletionGroupsDuringUpdate: h.ProviderConfiguration.DeletionGroupsDuringUpdate,
		InterruptionChecker:        interruption.NewStandardInterruptionChecker(h.DeployItem, h.lsKubeClient),
	})

	err := applier.Apply(ctx)
	h.ProviderStatus.ManagedResources = applier.GetManagedResourcesStatus()

	return err
}

func (h *Helm) createManifests(ctx context.Context, currOp string, files, crds map[string]string) ([]managedresource.Manifest, error) {
	logger, _ := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "createManifests"})

	if _, err := timeout.TimeoutExceeded(ctx, h.DeployItem, TimeoutCheckpointHelmStartCreateManifests); err != nil {
		return nil, err
	}

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
		t, lserr := timeout.TimeoutExceeded(ctx, h.DeployItem, TimeoutCheckpointHelmDefaultReadinessChecks)
		if lserr != nil {
			return lserr
		}

		defaultReadinessCheck := health.DefaultReadinessCheck{
			Context:             ctx,
			Client:              client,
			CurrentOp:           "DefaultCheckResourcesReadinessHelm",
			Timeout:             &lsv1alpha1.Duration{Duration: t},
			ManagedResources:    h.ProviderStatus.ManagedResources.TypedObjectReferenceList(),
			FailOnMissingObject: failOnMissingObject,
			InterruptionChecker: interruption.NewStandardInterruptionChecker(h.DeployItem, h.lsKubeClient),
		}
		err := defaultReadinessCheck.CheckResourcesReady()
		if err != nil {
			return err
		}
	}

	if h.ProviderConfiguration.ReadinessChecks.CustomReadinessChecks != nil {
		for _, customReadinessCheckConfig := range h.ProviderConfiguration.ReadinessChecks.CustomReadinessChecks {
			t, lserr := timeout.TimeoutExceeded(ctx, h.DeployItem, TimeoutCheckpointHelmCustomReadinessChecks)
			if lserr != nil {
				return lserr
			}
			customReadinessCheck := health.CustomReadinessCheck{
				Context:             ctx,
				Client:              client,
				CurrentOp:           "CustomCheckResourcesReadinessHelm",
				Timeout:             &lsv1alpha1.Duration{Duration: t},
				ManagedResources:    h.ProviderStatus.ManagedResources.TypedObjectReferenceList(),
				Configuration:       customReadinessCheckConfig,
				InterruptionChecker: interruption.NewStandardInterruptionChecker(h.DeployItem, h.lsKubeClient),
			}
			err := customReadinessCheck.CheckResourcesReady()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (h *Helm) readExportValues(ctx context.Context, currOp string, targetClient client.Client, exports map[string]interface{}) error {
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
			InterruptionChecker: interruption.NewStandardInterruptionChecker(h.DeployItem, h.lsKubeClient),
			DeployItem:          h.DeployItem,
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
		return h.deleteManifestsInGroups(ctx)
	} else {
		return h.deleteManifestsWithRealHelmDeployer(ctx, h.DeployItem)
	}
}

func (h *Helm) deleteManifestsInGroups(ctx context.Context) error {
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "deleteManifests"})
	logger.Info("Deleting files in groups")

	h.DeployItem.Status.Phase = lsv1alpha1.DeployItemPhases.Deleting

	if h.ProviderStatus == nil || len(h.ProviderStatus.ManagedResources) == 0 {
		controllerutil.RemoveFinalizer(h.DeployItem, lsv1alpha1.LandscaperFinalizer)
		return h.Writer().UpdateDeployItem(ctx, read_write_layer.W000067, h.DeployItem)
	}

	_, targetClient, _, err := h.TargetClient(ctx)
	if err != nil {
		return err
	}

	interruptionChecker := interruption.NewStandardInterruptionChecker(h.DeployItem, h.lsKubeClient)

	err = resourcemanager.DeleteManagedResources(
		ctx,
		h.ProviderStatus.ManagedResources,
		h.ProviderConfiguration.DeletionGroups,
		targetClient,
		h.DeployItem,
		interruptionChecker)
	if err != nil {
		return fmt.Errorf("failed deleting managed resources: %w", err)
	}

	// remove finalizer
	controllerutil.RemoveFinalizer(h.DeployItem, lsv1alpha1.LandscaperFinalizer)
	return h.Writer().UpdateDeployItem(ctx, read_write_layer.W000049, h.DeployItem)
}

func (h *Helm) deleteManifestsWithRealHelmDeployer(ctx context.Context, di *lsv1alpha1.DeployItem) error {
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
		h.TargetRestConfig, targetClientSet, di)

	err = realHelmDeployer.Undeploy(ctx)
	if err != nil {
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
