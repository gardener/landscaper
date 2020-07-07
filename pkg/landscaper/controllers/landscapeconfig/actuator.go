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

package landscapeconfig

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/pkg/utils"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
)

func NewActuator() (reconcile.Reconciler, error) {
	return &actuator{}, nil
}

type actuator struct {
	log    logr.Logger
	c      client.Client
	scheme *runtime.Scheme
}

var _ inject.Scheme = &actuator{}

// InjectClients injects the current kubernetes client into the actuator
func (a *actuator) InjectClient(c client.Client) error {
	a.c = c
	return nil
}

// InjectLogger injects a logging instance into the actuator
func (a *actuator) InjectLogger(log logr.Logger) error {
	a.log = log
	return nil
}

// InjectScheme injects the current scheme into the actuator
func (a *actuator) InjectScheme(scheme *runtime.Scheme) error {
	a.scheme = scheme
	return nil
}

func (a *actuator) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	defer ctx.Done()
	a.log.Info("reconcile", "resource", req.NamespacedName)

	lsConfig := &lsv1alpha1.LandscapeConfiguration{}
	if err := a.c.Get(ctx, req.NamespacedName, lsConfig); err != nil {
		a.log.Error(err, "unable to get resource")
		return reconcile.Result{}, err
	}

	if err := a.Ensure(ctx, lsConfig); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (a *actuator) Ensure(ctx context.Context, lsConfig *lsv1alpha1.LandscapeConfiguration) error {
	lsConfigData, err := a.reloadConfiguration(ctx, lsConfig)
	if err != nil {
		return err
	}

	if err := a.createOrUpdateConfigurationData(ctx, lsConfig, lsConfigData); err != nil {
		return err
	}

	cond := lsv1alpha1helper.GetOrInitCondition(lsConfig.Status.Conditions, lsv1alpha1.CollectReferencedConfiguration)
	cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue,
		"ConfigurationLoaded", "All configuration data has been successfully reloaded from the referenced secrets")
	if err := a.updateStatus(ctx, lsConfig, cond); err != nil {
		return err
	}

	return nil
}

func (a *actuator) updateStatus(ctx context.Context, lsConfig *lsv1alpha1.LandscapeConfiguration, updatedConditions ...lsv1alpha1.Condition) error {
	// todo: set export generation
	lsConfig.Status.Conditions = lsv1alpha1helper.MergeConditions(lsConfig.Status.Conditions, updatedConditions...)
	lsConfig.Status.ObservedGeneration = lsConfig.Generation
	if err := a.c.Status().Update(ctx, lsConfig); err != nil {
		a.log.Error(err, "unable to update landscape config status")
		return err
	}
	return nil
}

func (a *actuator) reloadConfiguration(ctx context.Context, lsConfig *lsv1alpha1.LandscapeConfiguration) ([]byte, error) {
	var (
		cond         = lsv1alpha1helper.GetOrInitCondition(lsConfig.Status.Conditions, lsv1alpha1.CollectReferencedConfiguration)
		lsConfigData = make(map[string]interface{})
	)

	for _, ref := range lsConfig.Spec.SecretReferences {
		secret := &corev1.Secret{}
		if err := a.c.Get(ctx, ref.NamespacedName(), secret); err != nil {
			a.log.Error(err, "unable to collect config from secret", "name", ref.Name, "namespace", ref.Namespace)
			cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				"ReferencedSecretNotFound", fmt.Sprintf("Referenced secret %s not available", ref.NamespacedName().String()))
			_ = a.updateStatus(ctx, lsConfig, cond)
			return nil, err
		}

		for key, data := range secret.Data {
			config := make(map[string]interface{})
			if err := yaml.Unmarshal(data, &config); err != nil {
				a.log.Error(err, "unable to parse config", "name", ref.Name, "namespace", ref.Namespace, "key", key)
				cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
					"ParseError", fmt.Sprintf("Config from referenced secret %s with key %s cannot be parsed", ref.NamespacedName().String(), key))
				_ = a.updateStatus(ctx, lsConfig, cond)
				return nil, err
			}

			lsConfigData = utils.MergeMaps(lsConfigData, config)
		}

		lsConfig.Status.Secrets = lsv1alpha1helper.CreateOrUpdateVersionedObjectReferences(lsConfig.Status.Secrets, ref, secret.Generation)
	}

	data, err := yaml.Marshal(lsConfigData)
	if err != nil {
		a.log.Error(err, "unable to encode config")
		cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			"EncodingError", "Merged config could not be encoded")
		_ = a.updateStatus(ctx, lsConfig, cond)
		return nil, err
	}

	return data, nil
}

// createOrUpdateConfigurationData takes the configuration and creates or updates a secret containing the data.
// This secret is then referenced in the configurations status.
func (a *actuator) createOrUpdateConfigurationData(ctx context.Context, lsConfig *lsv1alpha1.LandscapeConfiguration, lsConfigData []byte) error {
	cond := lsv1alpha1helper.GetOrInitCondition(lsConfig.Status.Conditions, lsv1alpha1.CollectReferencedConfiguration)

	secret := &corev1.Secret{}
	secret.GenerateName = lsConfig.Name
	secret.Namespace = lsConfig.Namespace

	if lsConfig.Status.ConfigReference != nil {
		secret.Name = lsConfig.Status.ConfigReference.Name
		secret.Namespace = lsConfig.Status.ConfigReference.Namespace
	}

	_, err := controllerutil.CreateOrUpdate(ctx, a.c, secret, func() error {
		secret.Data = map[string][]byte{
			lsv1alpha1.DataObjectSecretDataKey: lsConfigData,
		}
		if err := controllerutil.SetControllerReference(lsConfig, secret, a.scheme); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		a.log.Error(err, "unable to encode config")
		cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			"ConfigurationDataError", "Unable to update the exported configuration secret")
		_ = a.updateStatus(ctx, lsConfig, cond)
		return err
	}

	lsConfig.Status.ConfigReference = &lsv1alpha1.ObjectReference{Name: secret.Name, Namespace: secret.Namespace}
	return nil
}
