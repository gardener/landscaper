// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/credentials"
	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/utils/kubernetes"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helminstall "github.com/gardener/landscaper/apis/deployer/helm/install"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	helmv1alpha1validation "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1/validation"
	"github.com/gardener/landscaper/pkg/deployer/helm/chartresolver"
	"github.com/gardener/landscaper/pkg/utils"

	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

const (
	Type lsv1alpha1.DeployItemType = "landscaper.gardener.cloud/helm"
)

var HelmScheme = runtime.NewScheme()

func init() {
	helminstall.Install(HelmScheme)
}

// NewDeployItemBuilder creates a new deployitem builder for helm deployitems
func NewDeployItemBuilder() *utils.DeployItemBuilder {
	return utils.NewDeployItemBuilder(string(Type)).Scheme(HelmScheme)
}

// Helm is the internal representation of a DeployItem of Type Helm
type Helm struct {
	log           logr.Logger
	kubeClient    client.Client
	Configuration *helmv1alpha1.Configuration

	DeployItem            *lsv1alpha1.DeployItem
	Target                *lsv1alpha1.Target
	ProviderConfiguration *helmv1alpha1.ProviderConfiguration
	ProviderStatus        *helmv1alpha1.ProviderStatus
	componentsRegistryMgr *componentsregistry.Manager
}

// New creates a new internal helm item
func New(log logr.Logger, helmconfig *helmv1alpha1.Configuration, kubeClient client.Client, item *lsv1alpha1.DeployItem, target *lsv1alpha1.Target, componentsRegistryMgr *componentsregistry.Manager) (*Helm, error) {
	currOp := "InitHelmOperation"
	config := &helmv1alpha1.ProviderConfiguration{}
	helmdecoder := api.NewDecoder(HelmScheme)
	if _, _, err := helmdecoder.Decode(item.Spec.Configuration.Raw, nil, config); err != nil {
		return nil, lsv1alpha1helper.NewWrappedError(err,
			currOp, "ParseProviderConfiguration", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
	}

	if err := helmv1alpha1validation.ValidateProviderConfiguration(config); err != nil {
		return nil, lsv1alpha1helper.NewWrappedError(err,
			currOp, "ValidateProviderConfiguration", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
	}

	var status *helmv1alpha1.ProviderStatus
	if item.Status.ProviderStatus != nil {
		status = &helmv1alpha1.ProviderStatus{}
		if _, _, err := helmdecoder.Decode(item.Status.ProviderStatus.Raw, nil, status); err != nil {
			return nil, lsv1alpha1helper.NewWrappedError(err,
				currOp, "ParseProviderStatus", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
		}
	}

	return &Helm{
		log:                   log.WithValues("deployitem", kutil.ObjectKey(item.Name, item.Namespace)),
		kubeClient:            kubeClient,
		Configuration:         helmconfig,
		DeployItem:            item,
		Target:                target,
		ProviderConfiguration: config,
		ProviderStatus:        status,
		componentsRegistryMgr: componentsRegistryMgr,
	}, nil
}

// Template loads the specified helm chart
// and templates it with the given values.
func (h *Helm) Template(ctx context.Context) (map[string]string, map[string]interface{}, error) {
	currOp := "TemplateChart"

	restConfig, _, err := h.TargetClient()
	if err != nil {
		return nil, nil, lsv1alpha1helper.NewWrappedError(err,
			currOp, "GetTargetClient", err.Error())
	}

	// download chart
	// todo: do caching of charts
	ociClient, err := createOCIClient(ctx, h.log, h.kubeClient, h.DeployItem, h.Configuration, h.componentsRegistryMgr)
	if err != nil {
		return nil, nil, lsv1alpha1helper.NewWrappedError(err,
			currOp, "BuildOCIClient", err.Error())
	}
	ch, err := chartresolver.GetChart(ctx, h.log.WithName("chartresolver"), ociClient, &h.ProviderConfiguration.Chart)
	if err != nil {
		return nil, nil, lsv1alpha1helper.NewWrappedError(err,
			currOp, "GetHelmChart", err.Error())
	}

	//template chart
	options := chartutil.ReleaseOptions{
		Name:      h.ProviderConfiguration.Name,
		Namespace: h.ProviderConfiguration.Namespace,
		Revision:  0,
		IsInstall: true,
	}

	values := make(map[string]interface{})
	if err := yaml.Unmarshal(h.ProviderConfiguration.Values, &values); err != nil {
		return nil, nil, lsv1alpha1helper.NewWrappedError(err,
			currOp, "ParseHelmValues", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
	}
	values, err = chartutil.ToRenderValues(ch, values, options, nil)
	if err != nil {
		return nil, nil, lsv1alpha1helper.NewWrappedError(err,
			currOp, "RenderHelmValues", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
	}

	files, err := engine.RenderWithClient(ch, values, restConfig)
	if err != nil {
		return nil, nil, lsv1alpha1helper.NewWrappedError(err,
			currOp, "RenderHelmValues", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
	}

	return files, values, nil
}

func (h *Helm) TargetClient() (*rest.Config, client.Client, error) {
	// use the configured kubeconfig over the target if defined
	if len(h.ProviderConfiguration.Kubeconfig) != 0 {
		kubeconfig, err := base64.StdEncoding.DecodeString(h.ProviderConfiguration.Kubeconfig)
		if err != nil {
			return nil, nil, err
		}
		cConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
		if err != nil {
			return nil, nil, err
		}
		restConfig, err := cConfig.ClientConfig()
		if err != nil {
			return nil, nil, err
		}

		kubeClient, err := client.New(restConfig, client.Options{})
		if err != nil {
			return nil, nil, err
		}
		return restConfig, kubeClient, nil
	}
	if h.Target != nil {
		targetConfig := &lsv1alpha1.KubernetesClusterTargetConfig{}
		if err := json.Unmarshal(h.Target.Spec.Configuration.RawMessage, targetConfig); err != nil {
			return nil, nil, fmt.Errorf("unable to parse target confíguration: %w", err)
		}
		kubeconfig, err := clientcmd.NewClientConfigFromBytes([]byte(targetConfig.Kubeconfig))
		if err != nil {
			return nil, nil, err
		}
		restConfig, err := kubeconfig.ClientConfig()
		if err != nil {
			return nil, nil, err
		}

		kubeClient, err := client.New(restConfig, client.Options{})
		if err != nil {
			return nil, nil, err
		}
		return restConfig, kubeClient, nil
	}
	return nil, nil, errors.New("neither a target nor kubeconfig are defined")
}

func createOCIClient(ctx context.Context, log logr.Logger, client client.Client, item *lsv1alpha1.DeployItem, config *helmv1alpha1.Configuration, componentsRegistryMgr *componentsregistry.Manager) (ociclient.Client, error) {
	// resolve all pull secrets
	secrets, err := kubernetes.ResolveSecrets(ctx, client, item.Spec.RegistryPullSecrets)
	if err != nil {
		return nil, err
	}

	// always add a oci client to support unauthenticated requests
	ociConfigFiles := make([]string, 0)
	if config.OCI != nil {
		ociConfigFiles = config.OCI.ConfigFiles
	}
	ociKeyring, err := credentials.CreateOCIRegistryKeyring(secrets, ociConfigFiles)
	if err != nil {
		return nil, err
	}
	ociClient, err := ociclient.NewClient(log,
		utils.WithConfiguration(config.OCI),
		ociclient.WithKeyring(ociKeyring),
		ociclient.WithCache(componentsRegistryMgr.SharedCache()),
	)
	if err != nil {
		return nil, err
	}

	return ociClient, nil
}
