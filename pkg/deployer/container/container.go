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

package helm

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	containerinstall "github.com/gardener/landscaper/pkg/apis/deployer/container/install"
	containerv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/container/v1alpha1"
	container1alpha1validation "github.com/gardener/landscaper/pkg/apis/deployer/container/v1alpha1/validation"
	"github.com/gardener/landscaper/pkg/landscaper/registry"
)

const (
	Type lsv1alpha1.ExecutionType = "Container"
)

var Scheme = runtime.NewScheme()

func init() {
	containerinstall.Install(Scheme)
}

// Container is the internal representation of a DeployItem of Type Container
type Container struct {
	log            logr.Logger
	kubeClient     client.Client
	registryClient *registry.Registry

	DeployItem            *lsv1alpha1.DeployItem
	ProviderConfiguration *containerv1alpha1.ProviderConfiguration
}

// New creates a new internal helm item
func New(log logr.Logger, kubeClient client.Client, client *registry.Registry, config *containerv1alpha1.Configuration, item *lsv1alpha1.DeployItem) (*Container, error) {
	providerConfig := &containerv1alpha1.ProviderConfiguration{}
	decoder := serializer.NewCodecFactory(Scheme).UniversalDecoder()
	if _, _, err := decoder.Decode(item.Spec.Configuration, nil, providerConfig); err != nil {
		return nil, err
	}

	applyDefaults(config, providerConfig)

	if err := container1alpha1validation.ValidateProviderConfiguration(providerConfig); err != nil {
		return nil, err
	}

	return &Container{
		log:                   log,
		kubeClient:            kubeClient,
		registryClient:        client,
		DeployItem:            item,
		ProviderConfiguration: providerConfig,
	}, nil
}

func applyDefaults(config *containerv1alpha1.Configuration, providerConfig *containerv1alpha1.ProviderConfiguration) {
	if len(providerConfig.Image) == 0 {
		providerConfig.Image = config.DefaultImage
	}
}
