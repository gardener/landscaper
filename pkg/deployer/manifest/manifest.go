// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lserrors "github.com/gardener/landscaper/apis/errors"

	"github.com/gardener/landscaper/pkg/deployer/lib"

	"github.com/gardener/landscaper/pkg/utils"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	manifestinstall "github.com/gardener/landscaper/apis/deployer/manifest/install"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"

	manifestvalidation "github.com/gardener/landscaper/apis/deployer/manifest/validation"
	"github.com/gardener/landscaper/pkg/api"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
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
	log            logr.Logger
	lsKubeClient   client.Client
	hostKubeClient client.Client
	Configuration  *manifestv1alpha2.Configuration

	DeployItem            *lsv1alpha1.DeployItem
	Target                *lsv1alpha1.Target
	ProviderConfiguration *manifestv1alpha2.ProviderConfiguration
	ProviderStatus        *manifestv1alpha2.ProviderStatus

	TargetKubeClient client.Client
	TargetRestConfig *rest.Config
}

// NewDeployItemBuilder creates a new deployitem builder for manifest deployitems
func NewDeployItemBuilder() *utils.DeployItemBuilder {
	return utils.NewDeployItemBuilder(string(Type)).Scheme(Scheme)
}

// New creates a new internal manifest item
func New(log logr.Logger,
	lsKubeClient client.Client,
	hostKubeClient client.Client,
	configuration *manifestv1alpha2.Configuration,
	item *lsv1alpha1.DeployItem,
	target *lsv1alpha1.Target) (*Manifest, error) {

	config := &manifestv1alpha2.ProviderConfiguration{}
	currOp := "InitManifestOperation"
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
		log:                   log.WithValues("deployitem", kutil.ObjectKey(item.Name, item.Namespace)),
		lsKubeClient:          lsKubeClient,
		hostKubeClient:        hostKubeClient,
		Configuration:         configuration,
		DeployItem:            item,
		Target:                target,
		ProviderConfiguration: config,
		ProviderStatus:        status,
	}, nil
}

func (m *Manifest) TargetClient(ctx context.Context) (*rest.Config, client.Client, error) {
	if m.TargetKubeClient != nil {
		return m.TargetRestConfig, m.TargetKubeClient, nil
	}
	// use the configured kubeconfig over the target if defined
	if len(m.ProviderConfiguration.Kubeconfig) != 0 {
		kubeconfig, err := base64.StdEncoding.DecodeString(m.ProviderConfiguration.Kubeconfig)
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
	if m.Target != nil {
		targetConfig := &lsv1alpha1.KubernetesClusterTargetConfig{}
		if err := json.Unmarshal(m.Target.Spec.Configuration.RawMessage, targetConfig); err != nil {
			return nil, nil, fmt.Errorf("unable to parse target confíguration: %w", err)
		}

		kubeconfigBytes, err := lib.GetKubeconfigFromTargetConfig(ctx, targetConfig, m.lsKubeClient, m.hostKubeClient)
		if err != nil {
			return nil, nil, err
		}

		kubeconfig, err := clientcmd.NewClientConfigFromBytes(kubeconfigBytes)
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
