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

package operation

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/dataobject"
	"github.com/gardener/landscaper/pkg/landscaper/landscapeconfig"
	"github.com/gardener/landscaper/pkg/landscaper/registry"
)

// Operation is the operation interface that is used to share common operational data across the landscaper reconciler.
type Interface interface {
	Log() logr.Logger
	Client() client.Client
	Scheme() *runtime.Scheme
	Registry() registry.Registry
	GetDataObjectFromSecret(ctx context.Context, key types.NamespacedName) (*dataobject.DataObject, error)

	GetLandscapeConfig(ctx context.Context, namespace string) (*landscapeconfig.LandscapeConfig, error)
}

type operation struct {
	log      logr.Logger
	client   client.Client
	scheme   *runtime.Scheme
	registry registry.Registry
}

// NewOperation creates a new internal installation operation object.
func NewOperation(log logr.Logger, c client.Client, scheme *runtime.Scheme, registry registry.Registry) Interface {
	return &operation{
		log:      log,
		client:   c,
		scheme:   scheme,
		registry: registry,
	}
}

// Log returns a logging instance
func (o *operation) Log() logr.Logger {
	return o.log
}

// Client returns a controller runtime client.Client
func (o *operation) Client() client.Client {
	return o.client
}

// Schema returns a kubernetes scheme
func (o *operation) Scheme() *runtime.Scheme {
	return o.scheme
}

// Registry returns a registry.Registry instance
func (o *operation) Registry() registry.Registry {
	return o.registry
}

// GetDataObjectFromSecret creates a dataobject from a secret
func (o *operation) GetDataObjectFromSecret(ctx context.Context, key types.NamespacedName) (*dataobject.DataObject, error) {
	secret := &corev1.Secret{}
	if err := o.Client().Get(ctx, key, secret); err != nil {
		return nil, err
	}

	do, err := dataobject.New(secret)
	if err != nil {
		return nil, err
	}

	return do, nil
}

// GetLandscapeConfig reads the current landscaper config of the given namespace from the cluster
func (o *operation) GetLandscapeConfig(ctx context.Context, namespace string) (*landscapeconfig.LandscapeConfig, error) {
	lsConfig := &lsv1alpha1.LandscapeConfiguration{}
	if err := o.Client().Get(ctx, client.ObjectKey{Name: lsv1alpha1.LandscapeConfigName, Namespace: namespace}, lsConfig); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	if !IsLandscapeConfigReady(lsConfig) {
		o.Log().V(3).Info("LandscapeConfiguration is not ready")
		return nil, nil
	}

	do, err := o.GetDataObjectFromSecret(ctx, lsConfig.Status.ConfigReference.NamespacedName())
	if err != nil {
		return nil, err
	}

	return &landscapeconfig.LandscapeConfig{
		Info: lsConfig,
		Data: do,
	}, nil
}

// IsLandscapeConfigReady validates if the landsacpe config is ready to be read
func IsLandscapeConfigReady(lsConfig *lsv1alpha1.LandscapeConfiguration) bool {
	if lsConfig.Status.ConfigReference == nil {
		return false
	}

	if lsConfig.Generation != lsConfig.Status.ObservedGeneration {
		return false
	}

	if !lsv1alpha1helper.IsConditionStatus(lsConfig.Status.Conditions, lsv1alpha1.ConditionTrue) {
		return false
	}

	return true
}
