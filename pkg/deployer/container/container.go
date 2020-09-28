// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	blueprintsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/blueprints"
)

const (
	Type lsv1alpha1.ExecutionType = "Container"
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
	kubeClient    client.Client
	registry      blueprintsregistry.Registry
	Configuration *containerv1alpha1.Configuration

	DeployItem            *lsv1alpha1.DeployItem
	ProviderStatus        *containerv1alpha1.ProviderStatus
	ProviderConfiguration *containerv1alpha1.ProviderConfiguration

	InitContainerServiceAccountSecret types.NamespacedName
	WaitContainerServiceAccountSecret types.NamespacedName
}

// New creates a new internal helm item
func New(log logr.Logger, kubeClient client.Client, client blueprintsregistry.Registry, config *containerv1alpha1.Configuration, item *lsv1alpha1.DeployItem) (*Container, error) {
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
		kubeClient:            kubeClient,
		registry:              client,
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
