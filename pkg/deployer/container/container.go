// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	containerinstall "github.com/gardener/landscaper/pkg/apis/deployer/container/install"
	containerv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/container/v1alpha1"
	container1alpha1validation "github.com/gardener/landscaper/pkg/apis/deployer/container/v1alpha1/validation"
)

const (
	Type lsv1alpha1.DeployItemType = "landscaper.gardener.cloud/container"
)

var (
	Scheme = runtime.NewScheme()
)

func init() {
	containerinstall.Install(Scheme)
}

// Container is the internal representation of a DeployItem of Type Container
type Container struct {
	log           logr.Logger
	lsClient      client.Client
	hostClient    client.Client
	Configuration *containerv1alpha1.Configuration

	DeployItem            *lsv1alpha1.DeployItem
	ProviderStatus        *containerv1alpha1.ProviderStatus
	ProviderConfiguration *containerv1alpha1.ProviderConfiguration

	InitContainerServiceAccountSecret types.NamespacedName
	WaitContainerServiceAccountSecret types.NamespacedName
}

// New creates a new internal container item
func New(log logr.Logger, lsClient, hostClient client.Client, config *containerv1alpha1.Configuration, item *lsv1alpha1.DeployItem) (*Container, error) {
	providerConfig := &containerv1alpha1.ProviderConfiguration{}
	decoder := serializer.NewCodecFactory(Scheme).UniversalDecoder()
	if _, _, err := decoder.Decode(item.Spec.Configuration.Raw, nil, providerConfig); err != nil {
		return nil, err
	}

	applyDefaults(config, providerConfig)

	if err := container1alpha1validation.ValidateProviderConfiguration(providerConfig); err != nil {
		return nil, err
	}

	status, err := DecodeProviderStatus(item.Status.ProviderStatus)
	if err != nil {
		return nil, err
	}

	return &Container{
		log:                   log,
		lsClient:              lsClient,
		hostClient:            hostClient,
		Configuration:         config,
		DeployItem:            item,
		ProviderStatus:        status,
		ProviderConfiguration: providerConfig,
	}, nil
}

func applyDefaults(config *containerv1alpha1.Configuration, providerConfig *containerv1alpha1.ProviderConfiguration) {
	if len(providerConfig.Image) == 0 {
		providerConfig.Image = config.DefaultImage.Image
	}
}
