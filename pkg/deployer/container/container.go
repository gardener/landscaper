// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"github.com/gardener/component-cli/ociclient/cache"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lserrors "github.com/gardener/landscaper/apis/errors"

	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/utils"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containerinstall "github.com/gardener/landscaper/apis/deployer/container/install"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	container1alpha1validation "github.com/gardener/landscaper/apis/deployer/container/v1alpha1/validation"
)

const (
	Type lsv1alpha1.DeployItemType = "landscaper.gardener.cloud/container"
	Name string                    = "container.deployer.landscaper.gardener.cloud"
)

var (
	Scheme = runtime.NewScheme()
)

func init() {
	containerinstall.Install(Scheme)
}

// NewDeployItemBuilder creates a new deployitem builder for container deployitems
func NewDeployItemBuilder() *utils.DeployItemBuilder {
	return utils.NewDeployItemBuilder(string(Type)).Scheme(Scheme)
}

// Container is the internal representation of a DeployItem of Type Container
type Container struct {
	lsUncachedClient   client.Client
	lsCachedClient     client.Client
	hostUncachedClient client.Client
	hostCachedClient   client.Client

	Configuration containerv1alpha1.Configuration

	DeployItem            *lsv1alpha1.DeployItem
	Context               *lsv1alpha1.Context
	ProviderStatus        *containerv1alpha1.ProviderStatus
	ProviderConfiguration *containerv1alpha1.ProviderConfiguration
	Target                *lsv1alpha1.ResolvedTarget

	InitContainerServiceAccountSecret types.NamespacedName
	WaitContainerServiceAccountSecret types.NamespacedName

	sharedCache cache.Cache
}

// New creates a new internal container item
func New(lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient client.Client,
	config containerv1alpha1.Configuration,
	item *lsv1alpha1.DeployItem,
	lsCtx *lsv1alpha1.Context,
	sharedCache cache.Cache,
	rt *lsv1alpha1.ResolvedTarget) (*Container, error) {

	currOp := "InitContainerOperation"

	providerConfig := &containerv1alpha1.ProviderConfiguration{}
	decoder := api.NewDecoder(Scheme)
	if _, _, err := decoder.Decode(item.Spec.Configuration.Raw, nil, providerConfig); err != nil {
		return nil, lserrors.NewWrappedError(err,
			currOp, "DecodeProviderConfiguration", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
	}

	applyDefaults(&config, providerConfig)

	if err := container1alpha1validation.ValidateProviderConfiguration(providerConfig); err != nil {
		return nil, lserrors.NewWrappedError(err,
			currOp, "ValidateProviderConfiguration", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
	}

	status, err := DecodeProviderStatus(item.Status.ProviderStatus)
	if err != nil {
		return nil, lserrors.NewWrappedError(err,
			currOp, "DecodeProviderStatus", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
	}

	return &Container{
		lsUncachedClient:      lsUncachedClient,
		lsCachedClient:        lsCachedClient,
		hostUncachedClient:    hostUncachedClient,
		hostCachedClient:      hostCachedClient,
		Configuration:         config,
		DeployItem:            item,
		Context:               lsCtx,
		ProviderStatus:        status,
		ProviderConfiguration: providerConfig,
		sharedCache:           sharedCache,
		Target:                rt,
	}, nil
}

func applyDefaults(config *containerv1alpha1.Configuration, providerConfig *containerv1alpha1.ProviderConfiguration) {
	DefaultConfiguration(config)

	// default provider configuration
	if len(providerConfig.Image) == 0 {
		providerConfig.Image = config.DefaultImage.Image
	}
}
