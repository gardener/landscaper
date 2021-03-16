// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package terraform

import (
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/apis/deployer/terraform"
	terraforminstall "github.com/gardener/landscaper/apis/deployer/terraform/install"
	terraformv1alpha1 "github.com/gardener/landscaper/apis/deployer/terraform/v1alpha1"
	terraformv1alpha1validation "github.com/gardener/landscaper/apis/deployer/terraform/validation"
	kutils "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

var TerraformScheme = runtime.NewScheme()

func init() {
	terraforminstall.Install(TerraformScheme)
}

// Terraform is the internal representation of a DeployItem of Type Terraform.
type Terraform struct {
	log logr.Logger
	// lsClient is the kubernetes client that talks to the landscaper cluster.
	lsClient client.Client
	// hostClient is the kubernetes client that talks to the host cluster where the terraform controller is running.
	hostClient     client.Client
	hostRestConfig *rest.Config

	DeployItem            *lsv1alpha1.DeployItem
	Target                *lsv1alpha1.Target
	Configuration         *terraformv1alpha1.Configuration
	ProviderConfiguration *terraformv1alpha1.ProviderConfiguration
	ProviderStatus        *terraformv1alpha1.ProviderStatus
}

// New creates a new internal terraform item.
func New(log logr.Logger, lsClient, hostClient client.Client, hostRestConfig *rest.Config, config *terraformv1alpha1.Configuration, item *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) (*Terraform, error) {
	providerConfig := &terraformv1alpha1.ProviderConfiguration{}
	terraformdecoder := serializer.NewCodecFactory(TerraformScheme).UniversalDecoder()
	if _, _, err := terraformdecoder.Decode(item.Spec.Configuration.Raw, nil, providerConfig); err != nil {
		return nil, lsv1alpha1helper.NewWrappedError(err,
			"Init", "ParseProviderConfig", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
	}

	coreProviderConfig := &terraform.ProviderConfiguration{}
	if err := terraformv1alpha1.Convert_v1alpha1_ProviderConfiguration_To_terraform_ProviderConfiguration(providerConfig, coreProviderConfig, nil); err != nil {
		err = fmt.Errorf("failed to convert provider config to internal config: %w", err)
		return nil, lsv1alpha1helper.NewWrappedError(err,
			"Init", "ParseProviderConfig", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
	}
	if err := terraformv1alpha1validation.ValidateProviderConfiguration(coreProviderConfig); err != nil {
		return nil, lsv1alpha1helper.NewWrappedError(err,
			"Init", "ValidateProviderConfig", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
	}

	var status *terraformv1alpha1.ProviderStatus
	if item.Status.ProviderStatus != nil {
		status = &terraformv1alpha1.ProviderStatus{}
		if _, _, err := terraformdecoder.Decode(item.Status.ProviderStatus.Raw, nil, status); err != nil {
			return nil, lsv1alpha1helper.NewWrappedError(err,
				"Init", "ParseProviderStatus", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
		}
	}

	return &Terraform{
		log:            log.WithValues("resource", kutils.ObjectKey(item.Name, item.Namespace)),
		lsClient:       lsClient,
		hostClient:     hostClient,
		hostRestConfig: hostRestConfig,

		DeployItem:            item,
		Target:                target,
		Configuration:         config,
		ProviderConfiguration: providerConfig,
		ProviderStatus:        status,
	}, nil
}
