// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lserrors "github.com/gardener/landscaper/apis/errors"

	"github.com/gardener/landscaper/pkg/deployer/lib"

	"github.com/gardener/landscaper/pkg/utils"

	"k8s.io/apimachinery/pkg/util/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"
	manifestinstall "github.com/gardener/landscaper/apis/deployer/manifest/install"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"

	manifestvalidation "github.com/gardener/landscaper/apis/deployer/manifest/validation"
	"github.com/gardener/landscaper/pkg/api"
)

const (
	Type lsv1alpha1.DeployItemType = "landscaper.gardener.cloud/kubernetes-manifest"
	Name string                    = "manifest.deployer.landscaper.gardener.cloud"
)

var Scheme = runtime.NewScheme()

func init() {
	manifestinstall.Install(Scheme)
}

// Manifest is the internal representation of a DeployItem of Type Manifest
type Manifest struct {
	lsKubeClient   client.Client
	hostKubeClient client.Client
	Configuration  *manifestv1alpha2.Configuration

	DeployItem            *lsv1alpha1.DeployItem
	Target                *lsv1alpha1.ResolvedTarget
	ProviderConfiguration *manifestv1alpha2.ProviderConfiguration
	ProviderStatus        *manifestv1alpha2.ProviderStatus

	TargetKubeClient client.Client
	TargetRestConfig *rest.Config
	TargetClientSet  kubernetes.Interface
}

// NewDeployItemBuilder creates a new deployitem builder for manifest deployitems
func NewDeployItemBuilder() *utils.DeployItemBuilder {
	return utils.NewDeployItemBuilder(string(Type)).Scheme(Scheme)
}

// New creates a new internal manifest item
func New(lsKubeClient client.Client,
	hostKubeClient client.Client,
	configuration *manifestv1alpha2.Configuration,
	item *lsv1alpha1.DeployItem,
	rt *lsv1alpha1.ResolvedTarget) (*Manifest, error) {

	currOp := "InitManifestOperation"

	config := &manifestv1alpha2.ProviderConfiguration{}

	manifestDecoder := api.NewDecoder(Scheme)
	if _, _, err := manifestDecoder.Decode(item.Spec.Configuration.Raw, nil, config); err != nil {
		return nil, lserrors.NewWrappedError(err,
			currOp, "ParseProviderConfiguration", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
	}

	if err := manifestvalidation.ValidateProviderConfiguration(config); err != nil {
		return nil, lserrors.NewWrappedError(err,
			currOp, "ValidateProviderConfiguration", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
	}

	var status *manifestv1alpha2.ProviderStatus
	if item.Status.ProviderStatus != nil {
		status = &manifestv1alpha2.ProviderStatus{}
		if _, _, err := manifestDecoder.Decode(item.Status.ProviderStatus.Raw, nil, status); err != nil {
			return nil, lserrors.NewWrappedError(err,
				currOp, "ParseProviderStatus", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
		}
	}

	return &Manifest{
		lsKubeClient:          lsKubeClient,
		hostKubeClient:        hostKubeClient,
		Configuration:         configuration,
		DeployItem:            item,
		Target:                rt,
		ProviderConfiguration: config,
		ProviderStatus:        status,
	}, nil
}

func (m *Manifest) TargetClient(ctx context.Context) (*rest.Config, client.Client, kubernetes.Interface, error) {
	if m.TargetKubeClient != nil {
		return m.TargetRestConfig, m.TargetKubeClient, m.TargetClientSet, nil
	}
	// use the configured kubeconfig over the target if defined
	if len(m.ProviderConfiguration.Kubeconfig) != 0 {
		kubeconfig, err := base64.StdEncoding.DecodeString(m.ProviderConfiguration.Kubeconfig)
		if err != nil {
			return nil, nil, nil, err
		}
		cConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
		if err != nil {
			return nil, nil, nil, err
		}
		restConfig, err := cConfig.ClientConfig()
		if err != nil {
			return nil, nil, nil, err
		}

		kubeClient, err := client.New(restConfig, client.Options{})
		if err != nil {
			return nil, nil, nil, err
		}

		clientset, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return nil, nil, nil, err
		}

		m.TargetRestConfig = restConfig
		m.TargetKubeClient = kubeClient
		return restConfig, kubeClient, clientset, nil
	}
	if m.Target != nil {
		targetConfig := &targettypes.KubernetesClusterTargetConfig{}
		if err := yaml.Unmarshal([]byte(m.Target.Content), targetConfig); err != nil {
			return nil, nil, nil, fmt.Errorf("unable to parse target confíguration: %w", err)
		}

		kubeconfigBytes, err := lib.GetKubeconfigFromTargetConfig(ctx, targetConfig, m.Target.Namespace, m.lsKubeClient)
		if err != nil {
			return nil, nil, nil, err
		}

		kubeconfig, err := clientcmd.NewClientConfigFromBytes(kubeconfigBytes)
		if err != nil {
			return nil, nil, nil, err
		}
		restConfig, err := kubeconfig.ClientConfig()
		if err != nil {
			return nil, nil, nil, err
		}

		kubeClient, err := client.New(restConfig, client.Options{})
		if err != nil {
			return nil, nil, nil, err
		}
		clientset, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return nil, nil, nil, err
		}

		m.TargetRestConfig = restConfig
		m.TargetKubeClient = kubeClient
		m.TargetClientSet = clientset
		return restConfig, kubeClient, clientset, nil
	}
	return nil, nil, nil, errors.New("neither a target nor kubeconfig are defined")
}
