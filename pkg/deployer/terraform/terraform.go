// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package terraform

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gardener/gardener/extensions/pkg/terraformer"
	"github.com/gardener/gardener/pkg/logger"
	"github.com/gardener/landscaper/pkg/utils/kubernetes"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	terraforminstall "github.com/gardener/landscaper/pkg/apis/deployer/terraform/install"
	terraformv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/terraform/v1alpha1"
	terraformv1alpha1validation "github.com/gardener/landscaper/pkg/apis/deployer/terraform/v1alpha1/validation"
	corev1 "k8s.io/api/core/v1"
)

const (
	// Type defines the type of the execution.
	Type lsv1alpha1.ExecutionType = "landscaper.gardener.cloud/terraform"

	// TerraformerPurpose is a constant for the complete Terraform setup.
	TerraformerPurpose = "landscaper"

	// TerraformStateOutputsKey is the key to retrieve the ouputs from the state.
	TerraformStateOutputsKey = "outputs"
)

var TerraformScheme = runtime.NewScheme()

func init() {
	terraforminstall.Install(TerraformScheme)
}

// Terraform is the internal representation of a DeployItem of Type Terraform
type Terraform struct {
	log        logr.Logger
	kubeClient client.Client

	DeployItem            *lsv1alpha1.DeployItem
	Target                *lsv1alpha1.Target
	ProviderConfiguration *terraformv1alpha1.ProviderConfiguration
	ProviderStatus        *terraformv1alpha1.ProviderStatus
}

// New creates a new internal terraform item
func New(log logr.Logger, kubeClient client.Client, item *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) (*Terraform, error) {
	config := &terraformv1alpha1.ProviderConfiguration{}
	terraformdecoder := serializer.NewCodecFactory(TerraformScheme).UniversalDecoder()
	if _, _, err := terraformdecoder.Decode(item.Spec.Configuration.Raw, nil, config); err != nil {
		return nil, err
	}

	if err := terraformv1alpha1validation.ValidateProviderConfiguration(config); err != nil {
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
		log:        log,
		kubeClient: kubeClient,

		DeployItem:            item,
		Target:                target,
		ProviderConfiguration: config,
		ProviderStatus:        status,
	}, nil
}

// TargetClient returns the appropriate kubernetes config and client to use
// if a Kubeconfig or a Target was provided.
func (t *Terraform) TargetClient() (*rest.Config, client.Client, error) {
	// use the configured kubeconfig over the target if defined
	if len(t.ProviderConfiguration.Kubeconfig) != 0 {
		kubeconfig, err := base64.StdEncoding.DecodeString(t.ProviderConfiguration.Kubeconfig)
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
	if t.Target != nil {
		targetConfig := &lsv1alpha1.KubernetesClusterTargetConfig{}
		if err := json.Unmarshal(t.Target.Spec.Configuration, targetConfig); err != nil {
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

// NewTerraformer initializes a new Terraformer.
func NewTerraformer(restConfig *rest.Config, namespace, name, image, purpose string) (terraformer.Terraformer, error) {
	tfr, err := terraformer.NewForConfig(logger.NewLogger("info"), restConfig, purpose, namespace, name, image)
	if err != nil {
		return nil, err
	}

	return tfr.
		SetTerminationGracePeriodSeconds(630).
		SetDeadlineCleaning(5 * time.Minute).
		SetDeadlinePod(15 * time.Minute), nil
}

// SetTerraformerEnvVars configures the terraformer to use environment variables from secrets.
func SetTerraformerEnvVars(ctx context.Context, client client.Client, tfr terraformer.Terraformer, config *terraformv1alpha1.ProviderConfiguration) (terraformer.Terraformer, error) {
	envVars, err := computeTerraformerEnvVars(ctx, client, config)
	if err != nil {
		return nil, err
	}

	return tfr.SetEnvVars(envVars...), nil
}

// computeTerraformerEnvVars returns the EnvVar from the secrets in the provider configuration.
func computeTerraformerEnvVars(ctx context.Context, client client.Client, config *terraformv1alpha1.ProviderConfiguration) ([]corev1.EnvVar, error) {
	var envVars []corev1.EnvVar

	for _, name := range config.EnvSecrets {
		secret := &corev1.Secret{}
		if err := client.Get(ctx, kubernetes.ObjectKey(name, config.Namespace), secret); err != nil {
			return nil, err
		}

		if secret.Type != corev1.SecretTypeOpaque {
			return nil, fmt.Errorf("secret \"%s\" is not of the type Opaque: %s", name, secret.Type)
		}

		for key, _ := range secret.Data {
			envVars = append(envVars, corev1.EnvVar{
				Name: key,
				ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: name,
					},
					Key: key,
				}},
			})
		}
	}

	return envVars, nil
}
