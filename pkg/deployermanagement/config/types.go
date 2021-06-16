// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
)

var NoConfigError = errors.New("NO_CONFIG_ERROR")

// Internal Deployer types
// This is a copy of the types configured in deployers.
// This will be removed in the future when all deployers are completly externalized.
const (
	ContainerDeployerType = "landscaper.gardener.cloud/container"
	HelmDeployerType      = "landscaper.gardener.cloud/helm"
	ManifestDeployerType  = "landscaper.gardener.cloud/kubernetes-manifest"
	MockDeployerType      = "landscaper.gardener.cloud/mock"
)

// DeployerApplyArgs describes the configuration for the deployer apply function
type DeployerApplyArgs struct {
	Registration                *lsv1alpha1.DeployerRegistration
	Type                        lsv1alpha1.DeployItemType
	ComponentName, ResourceName string
	Values                      map[string]interface{}
}

type DeployerApplyFunc func(args DeployerApplyArgs) error

var DefaultDeployerConfiguration = map[string]DeployerApplyArgs{
	"container": {
		Type:          ContainerDeployerType,
		ComponentName: "github.com/gardener/landscaper/container-deployer",
		ResourceName:  "container-deployer-blueprint",
	},
	"helm": {
		Type:          HelmDeployerType,
		ComponentName: "github.com/gardener/landscaper/helm-deployer",
		ResourceName:  "helm-deployer-blueprint",
	},
	"manifest": {
		Type:          ManifestDeployerType,
		ComponentName: "github.com/gardener/landscaper/manifest-deployer",
		ResourceName:  "manifest-deployer-blueprint",
	},
	"mock": {
		Type:          MockDeployerType,
		ComponentName: "github.com/gardener/landscaper/mock-deployer",
		ResourceName:  "mock-deployer-blueprint",
	},
}

// DeployersConfiguration describes additional configuration for Deployers
type DeployersConfiguration struct {
	Deployers map[string]DeployerConfiguration `json:"Deployers"`
}

// DeployerConfigurationType describes the type of the deployer configuration
type DeployerConfigurationType int

const (
	ValuesType DeployerConfigurationType = iota
	DeployerRegistrationType
)

// DeployerConfiguration defines deployer configuration that can either be a
// values file for raw data or a complete deployer registration
type DeployerConfiguration struct {
	Type                 DeployerConfigurationType
	Values               map[string]interface{}
	DeployerRegistration *lsv1alpha1.DeployerRegistration
}

// IsValueType checks if the deployer configuration is of type "ValuesType".
func (cfg DeployerConfiguration) IsValueType() bool {
	return cfg.Type == ValuesType
}

// IsRegistrationType checks if the deployer configuration is of type "DeployerRegistrationType".
func (cfg DeployerConfiguration) IsRegistrationType() bool {
	return cfg.Type == DeployerRegistrationType
}

func (cfg *DeployerConfiguration) UnmarshalJSON(data []byte) error {
	meta := &metav1.TypeMeta{}
	if err := json.Unmarshal(data, meta); err != nil {
		return fmt.Errorf("unnable to parse metadata: %w", err)
	}
	if len(meta.APIVersion) != 0 && len(meta.Kind) != 0 {
		// expect that a deployer configuration is given
		deployerReg := &lsv1alpha1.DeployerRegistration{}
		if _, _, err := api.Decoder.Decode(data, nil, deployerReg); err != nil {
			return fmt.Errorf("unable to decode deployer configuration into a DeployerRegistration: %w", err)
		}

		*cfg = DeployerConfiguration{
			Type:                 DeployerRegistrationType,
			DeployerRegistration: deployerReg,
		}
		return nil
	}

	// treat the configuration as helm values
	var values map[string]interface{}
	if err := json.Unmarshal(data, &values); err != nil {
		return fmt.Errorf("unable to decode deployer configuration into values: %w", err)
	}
	*cfg = DeployerConfiguration{
		Type:   ValuesType,
		Values: values,
	}
	return nil
}

func (cfg *DeployerConfiguration) MarshalJSON() ([]byte, error) {
	if cfg == nil {
		return nil, nil
	}
	switch cfg.Type {
	case ValuesType, DeployerRegistrationType:
		return json.Marshal(cfg.Values)
	default:
		return nil, fmt.Errorf("unknown type")
	}
}
