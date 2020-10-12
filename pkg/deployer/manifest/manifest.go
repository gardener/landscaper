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

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	manifestinstall "github.com/gardener/landscaper/pkg/apis/deployer/manifest/install"
	manifestv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/manifest/v1alpha1"
)

const (
	Type lsv1alpha1.ExecutionType = "Manifest"
)

var ManifestScheme = runtime.NewScheme()

func init() {
	manifestinstall.Install(ManifestScheme)
}

// Manifest is the internal representation of a DeployItem of Type Manifest
type Manifest struct {
	log        logr.Logger
	kubeClient client.Client

	DeployItem            *lsv1alpha1.DeployItem
	Target                *lsv1alpha1.Target
	ProviderConfiguration *manifestv1alpha1.ProviderConfiguration
	ProviderStatus        *manifestv1alpha1.ProviderStatus
}

// New creates a new internal helm item
func New(log logr.Logger, kubeClient client.Client, item *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) (*Manifest, error) {
	config := &manifestv1alpha1.ProviderConfiguration{}
	manifestdecoder := serializer.NewCodecFactory(ManifestScheme).UniversalDecoder()
	if _, _, err := manifestdecoder.Decode(item.Spec.Configuration.Raw, nil, config); err != nil {
		return nil, err
	}

	var status *manifestv1alpha1.ProviderStatus
	if item.Status.ProviderStatus != nil {
		status = &manifestv1alpha1.ProviderStatus{}
		if _, _, err := manifestdecoder.Decode(item.Status.ProviderStatus.Raw, nil, status); err != nil {
			return nil, err
		}
	}

	return &Manifest{
		log:                   log,
		kubeClient:            kubeClient,
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
		if err := json.Unmarshal(m.Target.Spec.Configuration, targetConfig); err != nil {
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
