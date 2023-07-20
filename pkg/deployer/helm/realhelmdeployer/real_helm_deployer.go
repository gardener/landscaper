// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package realhelmdeployer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	apimachineryyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	lserror "github.com/gardener/landscaper/apis/errors"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/deployer/lib/readinesscheck"
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

type ManifestObject struct {
	APIVersion string               `json:"apiVersion"`
	Kind       string               `json:"kind"`
	ObjectMeta types.NamespacedName `json:"metadata"`

	// Items are needed for lists, e.g. ConfigMapLists.
	// Caution! The pointer is necessary to distinguish nil (ordinary object) from empty list.
	Items *[]ManifestObject `json:"items"`
}

func (o *ManifestObject) groupVersionKind() schema.GroupVersionKind {
	gv, err := schema.ParseGroupVersion(o.APIVersion)
	if err != nil {
		return schema.GroupVersionKind{}
	}
	return gv.WithKind(o.Kind)
}

func (o *ManifestObject) setDefaultNamespace(defaultNamespace string) *ManifestObject {
	if len(o.ObjectMeta.Namespace) == 0 {
		o.ObjectMeta.Namespace = defaultNamespace
	}
	return o
}

func (o *ManifestObject) toManagedResourceStatus() *managedresource.ManagedResourceStatus {
	return &managedresource.ManagedResourceStatus{
		Resource: corev1.ObjectReference{
			APIVersion: o.APIVersion,
			Kind:       o.Kind,
			Namespace:  o.ObjectMeta.Namespace,
			Name:       o.ObjectMeta.Name,
		},
	}
}

func (c *RealHelmDeployer) GetManagedResourcesStatus(ctx context.Context) ([]managedresource.ManagedResourceStatus, error) {
	release, err := c.getRelease(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]managedresource.ManagedResourceStatus, 0)
	reader := strings.NewReader(release.Manifest)
	decoder := apimachineryyaml.NewYAMLOrJSONDecoder(reader, 1024)
	for {
		obj := &ManifestObject{}
		if err := decoder.Decode(&obj); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("unable to decode item of helm manifest: %w", err)
		}

		if obj == nil {
			// Happens if the chart contains an empty template file, for example due to conditions.
			continue
		}

		if !readinesscheck.IsRelevantForDefaultReadinessCheck(obj.groupVersionKind().GroupKind()) {
			continue
		}

		// Caution! We must distinguish the case where obj.Items is nil (==> ordinary object, the most common case),
		// from the case where obj.Items is not nil but has length 0 (==> empty list).
		if obj.Items == nil {
			result = append(result, *obj.setDefaultNamespace(c.defaultNamespace).toManagedResourceStatus())
		} else {
			// expand object list
			items := *obj.Items
			for i := range items {
				item := &items[i]
				result = append(result, *item.setDefaultNamespace(c.defaultNamespace).toManagedResourceStatus())
			}
		}
	}

	return result, nil
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
	helmSecretManager, err := c.getHelmSecretManager(ctx)
	if err != nil {
		logger.Error(err, "get helm secret manager", lc.KeyResource, types.NamespacedName{Name: c.releaseName, Namespace: c.defaultNamespace}.String())
	} else {
		err = helmSecretManager.DeletePendingReleaseSecrets(ctx, c.releaseName)
		if err != nil {
			logger.Error(err, "delete helm secret", lc.KeyResource, types.NamespacedName{Name: c.releaseName, Namespace: c.defaultNamespace}.String())
		}
	}
}
