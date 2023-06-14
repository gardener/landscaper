// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package realhelmdeployer

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"strings"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"

	lserror "github.com/gardener/landscaper/apis/errors"

	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"

	"helm.sh/helm/v3/pkg/chart"

	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/client-go/rest"

	"helm.sh/helm/v3/pkg/action"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"

	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	lserrors "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/deployer/lib/resourcemanager"
)

type RealHelmDeployer struct {
	chart              *chart.Chart
	decoder            runtime.Decoder
	releaseName        string
	defaultNamespace   string
	rawValues          json.RawMessage
	helmConfig         *helmv1alpha1.HelmDeploymentConfiguration
	createNamespace    bool
	targetRestConfig   *rest.Config
	apiResourceHandler *resourcemanager.ApiResourceHandler
	helmSecretManager  *HelmSecretManager
}

func NewRealHelmDeployer(ch *chart.Chart, providerConfig *helmv1alpha1.ProviderConfiguration, targetRestConfig *rest.Config,
	clientset kubernetes.Interface) *RealHelmDeployer {

	return &RealHelmDeployer{
		chart:              ch,
		decoder:            serializer.NewCodecFactory(scheme.Scheme).UniversalDecoder(),
		releaseName:        providerConfig.Name,
		defaultNamespace:   providerConfig.Namespace,
		rawValues:          providerConfig.Values,
		helmConfig:         providerConfig.HelmDeploymentConfig,
		createNamespace:    providerConfig.CreateNamespace,
		targetRestConfig:   targetRestConfig,
		apiResourceHandler: resourcemanager.CreateApiResourceHandler(clientset),
		helmSecretManager:  nil,
	}
}

func (c *RealHelmDeployer) Deploy(ctx context.Context) error {
	values := make(map[string]interface{})
	if err := yaml.Unmarshal(c.rawValues, &values); err != nil {
		return lserrors.NewWrappedError(
			err, "Deploy", "ParseHelmValues", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
	}

	_, err := c.getRelease(ctx)
	if err != nil && c.isReleaseNotFoundErr(err) {
		_, err = c.installRelease(ctx, values)
		return err
	} else if err != nil {
		return err
	} else {
		_, err = c.upgradeRelease(ctx, values)
		return err
	}
}

func (c *RealHelmDeployer) Undeploy(ctx context.Context) error {
	return c.deleteRelease(ctx)
}

func (c *RealHelmDeployer) getRelease(ctx context.Context) (*release.Release, error) {
	currOp := "GetHelmRelease"

	actionConfig, err := c.initActionConfig(ctx)
	if err != nil {
		return nil, err
	}

	rls, err := action.NewGet(actionConfig).Run(c.releaseName)
	if err != nil {
		return nil, lserror.NewWrappedError(err, currOp, "GetRelease", err.Error())
	}

	// We check that the release found is from the provided namespace.
	// If `namespace` is an empty string we do not do that check
	// This check is to prevent users of for example updating releases that might be
	// in namespaces that they do not have access to.
	if c.defaultNamespace != "" && rls.Namespace != c.defaultNamespace {
		err := fmt.Errorf("release %q not found in namespace %q", c.releaseName, c.defaultNamespace)
		return nil, lserror.NewWrappedError(err, currOp, "CheckNamespace", err.Error())
	}

	return rls, err
}

// installRelease creates a helm release
func (c *RealHelmDeployer) installRelease(ctx context.Context, values map[string]interface{}) (*release.Release, error) {
	currOp := "InstallHelmRelease"
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, currOp})

	logger.Info(fmt.Sprintf("installing release %s into namespace %s", c.releaseName, c.defaultNamespace))

	actionConfig, err := c.initActionConfig(ctx)
	if err != nil {
		return nil, err
	}

	installConfig, err := newInstallConfiguration(c.helmConfig)
	if err != nil {
		return nil, err
	}

	install := action.NewInstall(actionConfig)
	install.ReleaseName = c.releaseName
	install.Namespace = c.defaultNamespace
	install.CreateNamespace = c.createNamespace
	install.Atomic = installConfig.Atomic
	install.Timeout = installConfig.Timeout.Duration

	logger.Info(fmt.Sprintf("installing helm chart release %s", c.releaseName))

	rel, err := install.Run(c.chart, values)
	if err != nil {
		c.unblockPendingHelmRelease(ctx, logger)

		message := fmt.Sprintf("unable to install helm chart release: %s", err.Error())
		logger.Info(message)
		return nil, lserror.NewWrappedError(err, currOp, "Install", message)
	}

	logger.Info(fmt.Sprintf("%s successfully installed in %s", c.releaseName, c.defaultNamespace))

	return rel, nil
}

// upgradeRelease upgrades a helm release
func (c *RealHelmDeployer) upgradeRelease(ctx context.Context, values map[string]interface{}) (*release.Release, error) {
	currOp := "UpgradeHelmRelease"
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, currOp})

	logger.Info(fmt.Sprintf("upgrading release %s", c.releaseName))

	actionConfig, err := c.initActionConfig(ctx)
	if err != nil {
		return nil, err
	}

	upgradeConfig, err := newUpgradeConfiguration(c.helmConfig)
	if err != nil {
		return nil, err
	}

	upgrade := action.NewUpgrade(actionConfig)
	upgrade.Namespace = c.defaultNamespace
	upgrade.MaxHistory = 10
	upgrade.Atomic = upgradeConfig.Atomic
	upgrade.Timeout = upgradeConfig.Timeout.Duration

	logger.Info(fmt.Sprintf("upgrading helm chart release %s", c.releaseName))

	rel, err := upgrade.Run(c.releaseName, c.chart, values)
	if err != nil {
		c.unblockPendingHelmRelease(ctx, logger)

		message := fmt.Sprintf("unable to upgrade helm chart release: %s", err.Error())
		logger.Info(message)
		return nil, lserror.NewWrappedError(err, currOp, "Install", message)
	}

	logger.Info(fmt.Sprintf("%s successfully upgraded in %s", c.releaseName, c.defaultNamespace))

	return rel, nil
}

func (c *RealHelmDeployer) deleteRelease(ctx context.Context) error {
	currOp := "DeleteHelmRelease"
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, currOp})

	logger.Info(fmt.Sprintf("deleting release %s in namespace %s", c.releaseName, c.defaultNamespace))

	// Validate that the release actually belongs to the namespace
	_, err := c.getRelease(ctx)
	if err != nil {
		return err
	}

	actionConfig, err := c.initActionConfig(ctx)
	if err != nil {
		return err
	}

	uninstallConfig, err := newUninstallConfiguration(c.helmConfig)
	if err != nil {
		return err
	}

	uninstall := action.NewUninstall(actionConfig)
	uninstall.KeepHistory = false
	uninstall.Timeout = uninstallConfig.Timeout.Duration

	_, err = uninstall.Run(c.releaseName)
	if err != nil {
		err2 := fmt.Errorf("unable to delete helm chart release: %w", err)
		return lserror.NewWrappedError(err2, currOp, "Uninstall", err2.Error())
	}

	logger.Info(fmt.Sprintf("%s successfully deleted in %s", c.releaseName, c.defaultNamespace))

	return nil
}

func (c *RealHelmDeployer) initActionConfig(ctx context.Context) (*action.Configuration, error) {
	logf := c.createLogFunc(ctx)

	currOp := "InitHelmAction"

	restClientGetter := newRemoteRESTClientGetter(c.targetRestConfig, c.defaultNamespace)
	kc := kube.New(restClientGetter)
	kc.Log = logf

	clientset, err := kc.Factory.KubernetesClientSet()
	if err != nil {
		return nil, lserror.NewWrappedError(err, currOp, "GetKubernetesClientSet", err.Error())
	}

	store := c.getStorageType(ctx, clientset, c.defaultNamespace)

	actionConfig := action.Configuration{
		RESTClientGetter: restClientGetter,
		Releases:         store,
		KubeClient:       kc,
		Log:              logf,
	}

	return &actionConfig, nil
}

func (c *RealHelmDeployer) getStorageType(ctx context.Context, clientset *kubernetes.Clientset, namespace string) *storage.Storage {
	logf := c.createLogFunc(ctx)

	var store *storage.Storage
	switch os.Getenv("HELM_DRIVER") {
	case "secret", "secrets", "":
		d := driver.NewSecrets(clientset.CoreV1().Secrets(namespace))
		d.Log = logf
		store = storage.Init(d)
	case "configmap", "configmaps":
		d := driver.NewConfigMaps(clientset.CoreV1().ConfigMaps(namespace))
		d.Log = logf
		store = storage.Init(d)
	case "memory":
		d := driver.NewMemory()
		store = storage.Init(d)
	default:
		// Not sure what to do here.
		panic("Unknown driver in HELM_DRIVER: " + os.Getenv("HELM_DRIVER"))
	}
	return store
}

func (c *RealHelmDeployer) createLogFunc(ctx context.Context) func(format string, v ...interface{}) {
	logger, _ := logging.FromContextOrNew(ctx, nil)
	return func(format string, v ...interface{}) {
		logger.Info(fmt.Sprintf(format, v))
	}
}

func (c *RealHelmDeployer) GetManagedResourcesStatus(ctx context.Context,
	manifests []managedresource.Manifest) ([]managedresource.ManagedResourceStatus, error) {
	result := make([]managedresource.ManagedResourceStatus, 0)

	for i := range manifests {
		typeMeta := metav1.TypeMeta{}
		if err := json.Unmarshal(manifests[i].Manifest.Raw, &typeMeta); err != nil {
			return nil, fmt.Errorf("unable to parse type metadata: %w", err)
		}
		innerManifest := &resourcemanager.Manifest{
			TypeMeta: typeMeta,
			Policy:   manifests[i].Policy,
			Manifest: manifests[i].Manifest,
		}

		nextResourceStatus, err := c.getResourceStatus(ctx, innerManifest)
		if err != nil {
			return nil, fmt.Errorf("unable to fetch next resource status: %w", err)
		}

		result = append(result, *nextResourceStatus)
	}

	return result, nil
}

func (c *RealHelmDeployer) getResourceStatus(_ context.Context,
	manifest *resourcemanager.Manifest) (*managedresource.ManagedResourceStatus, error) {
	currOp := "GetResourceStatus"

	gvk := manifest.TypeMeta.GetObjectKind().GroupVersionKind().String()
	obj := &unstructured.Unstructured{}
	if _, _, err := c.decoder.Decode(manifest.Manifest.Raw, nil, obj); err != nil {
		err2 := fmt.Errorf("error while decoding manifest %s: %w", gvk, err)
		return nil, lserror.NewWrappedError(err2, currOp, "ParseManifest", err2.Error())
	}

	if len(c.defaultNamespace) != 0 && len(obj.GetNamespace()) == 0 {
		// need to default the namespace if it is not given, as some helmcharts
		// do not use ".Release.Namespace" and depend on the helm/kubectl defaulting.
		apiresource, err := c.apiResourceHandler.GetApiResource(manifest)
		if err != nil {
			return nil, err
		}
		// only default namespaced resources.
		if apiresource.Namespaced {
			obj.SetNamespace(c.defaultNamespace)
		}
	}

	mr := &managedresource.ManagedResourceStatus{
		Policy:   manifest.Policy,
		Resource: *kutil.CoreObjectReferenceFromUnstructuredObject(obj),
	}

	return mr, nil
}

func (c *RealHelmDeployer) isReleaseNotFoundErr(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "release: not found")
}

func (c *RealHelmDeployer) getHelmSecretManager(ctx context.Context) (*HelmSecretManager, error) {
	var err error

	if c.helmSecretManager == nil {
		c.helmSecretManager, err = NewHelmSecretManager(c.targetRestConfig, c.defaultNamespace, c.createLogFunc(ctx))

		if err != nil {
			return nil, fmt.Errorf("failed to create helm secret manager: %w", err)
		}
	}

	return c.helmSecretManager, nil
}

func (c *RealHelmDeployer) unblockPendingHelmRelease(ctx context.Context, logger logging.Logger) {
	helmSecretManager, err1 := c.getHelmSecretManager(ctx)
	if err1 != nil {
		logger.Error(err1, "get helm secret manager", lc.KeyResource, types.NamespacedName{Name: c.releaseName, Namespace: c.defaultNamespace}.String())
	} else {
		err1 = helmSecretManager.DeletePendingReleaseSecrets(ctx, c.releaseName)
		if err1 != nil {
			logger.Error(err1, "delete helm secret", lc.KeyResource, types.NamespacedName{Name: c.releaseName, Namespace: c.defaultNamespace}.String())
		}
	}
}
