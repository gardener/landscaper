// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/manifest"
	manifestinstall "github.com/gardener/landscaper/apis/deployer/manifest/install"

	manifestvalidation "github.com/gardener/landscaper/apis/deployer/manifest/validation"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

const (
	Type lsv1alpha1.DeployItemType = "landscaper.gardener.cloud/kubernetes-manifest"
)

var ManifestScheme = runtime.NewScheme()

func init() {
	manifestinstall.Install(ManifestScheme)
}

// Manifest is the internal representation of a DeployItem of Type Manifest
type Manifest struct {
	log           logr.Logger
	kubeClient    client.Client
	Configuration *manifest.Configuration

	DeployItem            *lsv1alpha1.DeployItem
	Target                *lsv1alpha1.Target
	ProviderConfiguration *manifest.ProviderConfiguration
	ProviderStatus        *manifest.ProviderStatus
}

// New creates a new internal helm item
func New(log logr.Logger, kubeClient client.Client, manifestConfig *manifest.Configuration, item *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) (*Manifest, error) {
	config := &manifest.ProviderConfiguration{}
	manifestDecoder := serializer.NewCodecFactory(ManifestScheme).UniversalDecoder()
	if _, _, err := manifestDecoder.Decode(item.Spec.Configuration.Raw, nil, config); err != nil {
		return nil, err
	}

	if err := manifestvalidation.ValidateProviderConfiguration(config); err != nil {
		return nil, err
	}

	var status *manifest.ProviderStatus
	if item.Status.ProviderStatus != nil {
		status = &manifest.ProviderStatus{}
		if _, _, err := manifestDecoder.Decode(item.Status.ProviderStatus.Raw, nil, status); err != nil {
			return nil, err
		}
	}

	return &Manifest{
		log:                   log.WithValues("deployitem", kutil.ObjectKey(item.Name, item.Namespace)),
		kubeClient:            kubeClient,
		Configuration:         manifestConfig,
		DeployItem:            item,
		Target:                target,
		ProviderConfiguration: config,
		ProviderStatus:        status,
	}, nil
}

func (m *Manifest) TargetClient() (*rest.Config, client.Client, error) {
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
