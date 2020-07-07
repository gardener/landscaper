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
	"context"
	"encoding/base64"
	"fmt"

	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/helm/registry"
)

const (
	Type lsv1alpha1.ExecutionType = "Helm"
)

// Helm is the internal representation of a DeployItem of Type Helm
type Helm struct {
	log            logr.Logger
	kubeClient     client.Client
	registryClient *registry.Client

	DeployItem    *lsv1alpha1.DeployItem
	Configuration *Configuration
}

// New creates a new internal helm item
func New(log logr.Logger, kubeClient client.Client, client *registry.Client, item *lsv1alpha1.DeployItem) (*Helm, error) {
	config := &Configuration{}
	if err := yaml.Unmarshal(item.Spec.Configuration, config); err != nil {
		return nil, err
	}

	if err := Validate(config); err != nil {
		return nil, err
	}

	return &Helm{
		log:            log,
		kubeClient:     kubeClient,
		registryClient: client,
		DeployItem:     item,
		Configuration:  config,
	}, nil
}

// Template loads the specified helm chart
// and templates it with the given values.
func (h *Helm) Template(ctx context.Context) (map[string]string, error) {
	restConfig, _, err := h.TargetClient()
	if err != nil {
		return nil, err
	}

	// download chart
	// todo: do caching of charts
	ch, err := h.registryClient.GetChart(ctx, fmt.Sprintf("%s:%s", h.Configuration.Repository, h.Configuration.Version))
	if err != nil {
		return nil, err
	}

	//template chart
	options := chartutil.ReleaseOptions{
		Name:      h.Configuration.Name,
		Namespace: h.Configuration.Namespace,
		Revision:  0,
		IsInstall: true,
	}

	values, err := chartutil.ToRenderValues(ch, h.Configuration.Values, options, nil)
	if err != nil {
		return nil, err
	}

	return engine.RenderWithClient(ch, values, restConfig)
}

func (h *Helm) TargetClient() (*rest.Config, client.Client, error) {
	kubeconfig, err := base64.StdEncoding.DecodeString(h.Configuration.Kubeconfig)
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
