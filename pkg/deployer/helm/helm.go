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

	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	helminstall "github.com/gardener/landscaper/pkg/apis/deployer/helm/install"
	helmv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/helm/v1alpha1"
	helmv1alpha1validation "github.com/gardener/landscaper/pkg/apis/deployer/helm/v1alpha1/validation"
	"github.com/gardener/landscaper/pkg/deployer/helm/registry"
)

const (
	Type lsv1alpha1.ExecutionType = "landscaper.gardener.cloud/helm"
)

var Helmscheme = runtime.NewScheme()

func init() {
	helminstall.Install(Helmscheme)
}

// Helm is the internal representation of a DeployItem of Type Helm
type Helm struct {
	log            logr.Logger
	kubeClient     client.Client
	registryClient *registry.Client

	DeployItem            *lsv1alpha1.DeployItem
	Target                *lsv1alpha1.Target
	ProviderConfiguration *helmv1alpha1.ProviderConfiguration
	ProviderStatus        *helmv1alpha1.ProviderStatus
}

// New creates a new internal helm item
func New(log logr.Logger, kubeClient client.Client, client *registry.Client, item *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) (*Helm, error) {
	config := &helmv1alpha1.ProviderConfiguration{}
	helmdecoder := serializer.NewCodecFactory(Helmscheme).UniversalDecoder()
	if _, _, err := helmdecoder.Decode(item.Spec.Configuration.Raw, nil, config); err != nil {
		return nil, err
	}

	if err := helmv1alpha1validation.ValidateProviderConfiguration(config); err != nil {
		return nil, err
	}

	var status *helmv1alpha1.ProviderStatus
	if item.Status.ProviderStatus != nil {
		status = &helmv1alpha1.ProviderStatus{}
		if _, _, err := helmdecoder.Decode(item.Status.ProviderStatus.Raw, nil, status); err != nil {
			return nil, err
		}
	}

	return &Helm{
		log:                   log,
		kubeClient:            kubeClient,
		registryClient:        client,
		DeployItem:            item,
		Target:                target,
		ProviderConfiguration: config,
		ProviderStatus:        status,
	}, nil
}

// Template loads the specified helm chart
// and templates it with the given values.
func (h *Helm) Template(ctx context.Context) (map[string]string, map[string]interface{}, error) {
	restConfig, _, err := h.TargetClient()
	if err != nil {
		return nil, nil, err
	}

	// download chart
	// todo: do caching of charts
	ch, err := h.registryClient.GetChart(ctx, h.ProviderConfiguration.Chart.Ref)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}
	values, err = chartutil.ToRenderValues(ch, values, options, nil)
	if err != nil {
		return nil, nil, err
	}

	files, err := engine.RenderWithClient(ch, values, restConfig)
	if err != nil {
		return nil, nil, err
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
		if err := json.Unmarshal(h.Target.Spec.Configuration, targetConfig); err != nil {
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
