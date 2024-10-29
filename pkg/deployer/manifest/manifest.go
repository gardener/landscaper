// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lserrors "github.com/gardener/landscaper/apis/errors"

	"github.com/gardener/landscaper/pkg/deployer/lib"

	"github.com/gardener/landscaper/pkg/utils"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
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
	lsUncachedClient   client.Client
	hostUncachedClient client.Client

	Configuration *manifestv1alpha2.Configuration

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
func New(lsUncachedClient client.Client, hostUncachedClient client.Client,
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
		lsUncachedClient:      lsUncachedClient,
		hostUncachedClient:    hostUncachedClient,
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
	if m.Target != nil {
		restConfig, kubeClient, clientset, err := lib.GetRestConfigAndClientAndClientSet(ctx, m.Target, m.lsUncachedClient)
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
