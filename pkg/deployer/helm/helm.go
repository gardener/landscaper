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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	helminstall "github.com/gardener/landscaper/pkg/apis/deployer/helm/install"
	helmv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/helm/v1alpha1"
	helmv1alpha1validation "github.com/gardener/landscaper/pkg/apis/deployer/helm/v1alpha1/validation"
	"github.com/gardener/landscaper/pkg/deployer/helm/registry"
)

const (
	Type lsv1alpha1.ExecutionType = "Helm"
)

var Helmscheme = runtime.NewScheme()

func init() {
	helminstall.Install(Helmscheme)
}

// Helm is the internal representation of a DeployItem of Type Helm
type Helm struct {
	log            logr.Logger
	kubeClient     client.Client
	registryClient *registry.Client

	DeployItem    *lsv1alpha1.DeployItem
	Configuration *helmv1alpha1.ProviderConfiguration
}

// New creates a new internal helm item
func New(log logr.Logger, kubeClient client.Client, client *registry.Client, item *lsv1alpha1.DeployItem) (*Helm, error) {
	config := &helmv1alpha1.ProviderConfiguration{}
	helmdecoder := serializer.NewCodecFactory(Helmscheme).UniversalDecoder()
	if _, _, err := helmdecoder.Decode(item.Spec.Configuration, nil, config); err != nil {
		return nil, err
	}

	if err := helmv1alpha1validation.ValidateProviderConfiguration(config); err != nil {
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
func (h *Helm) Template(ctx context.Context) (map[string]string, map[string]interface{}, error) {
	restConfig, _, err := h.TargetClient()
	if err != nil {
		return nil, nil, err
	}

	// download chart
	// todo: do caching of charts
	ch, err := h.registryClient.GetChart(ctx, fmt.Sprintf("%s:%s", h.Configuration.Repository, h.Configuration.Version))
	if err != nil {
		return nil, nil, err
	}

	//template chart
	options := chartutil.ReleaseOptions{
		Name:      h.Configuration.Name,
		Namespace: h.Configuration.Namespace,
		Revision:  0,
		IsInstall: true,
	}

	values := make(map[string]interface{})
	if err := yaml.Unmarshal(h.Configuration.Values, &values); err != nil {
		return nil, nil, err
	}
	values, err = chartutil.ToRenderValues(ch, values, options, nil)
	if err != nil {
		return nil, nil, err
	}

	files, err := engine.RenderWithClient(ch, values, restConfig)
	if err != nil {
		return nil, nil, err
	}

	return files, values, nil
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
