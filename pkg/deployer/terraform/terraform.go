// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package terraform

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	terraforminstall "github.com/gardener/landscaper/apis/deployer/terraform/install"
	terraformv1alpha1 "github.com/gardener/landscaper/apis/deployer/terraform/v1alpha1"
	terraformv1alpha1validation "github.com/gardener/landscaper/apis/deployer/terraform/v1alpha1/validation"
	kutils "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

var TerraformScheme = runtime.NewScheme()

func init() {
	terraforminstall.Install(TerraformScheme)
}

// Terraform is the internal representation of a DeployItem of Type Terraform.
type Terraform struct {
	log        logr.Logger
	kubeClient client.Client
	restConfig *rest.Config

	DeployItem            *lsv1alpha1.DeployItem
	Target                *lsv1alpha1.Target
	Configuration         *terraformv1alpha1.Configuration
	ProviderConfiguration *terraformv1alpha1.ProviderConfiguration
	ProviderStatus        *terraformv1alpha1.ProviderStatus
}

// New creates a new internal terraform item.
func New(log logr.Logger, kubeClient client.Client, restConfig *rest.Config, config *terraformv1alpha1.Configuration, item *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) (*Terraform, error) {
	providerConfig := &terraformv1alpha1.ProviderConfiguration{}
	terraformdecoder := serializer.NewCodecFactory(TerraformScheme).UniversalDecoder()
	if _, _, err := terraformdecoder.Decode(item.Spec.Configuration.Raw, nil, providerConfig); err != nil {
		return nil, err
	}
	if err := terraformv1alpha1validation.ValidateProviderConfiguration(providerConfig); err != nil {
		return nil, err
	}

	var status *terraformv1alpha1.ProviderStatus
	if item.Status.ProviderStatus != nil {
		status = &terraformv1alpha1.ProviderStatus{}
		if _, _, err := terraformdecoder.Decode(item.Status.ProviderStatus.Raw, nil, status); err != nil {
			return nil, err
		}
	}

	return &Terraform{
		log:        log.WithValues("resource", kutils.ObjectKey(item.Name, item.Namespace)),
		kubeClient: kubeClient,
		restConfig: restConfig,

		DeployItem:            item,
		Target:                target,
		Configuration:         config,
		ProviderConfiguration: providerConfig,
		ProviderStatus:        status,
	}, nil
}
