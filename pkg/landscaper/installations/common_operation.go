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

package installations

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/registry"
)

// Operation is the operation interface that is used to share common operational data across the installation reconciler
//
type Operation interface {
	Log() logr.Logger
	Client() client.Client
	Scheme() *runtime.Scheme
	Registry() registry.Registry
	UpdateInstallationStatus(ctx context.Context, inst *lsv1alpha1.ComponentInstallation, phase lsv1alpha1.ComponentInstallationPhase, updatedConditions ...lsv1alpha1.Condition) error
}

type operation struct {
	log      logr.Logger
	client   client.Client
	scheme   *runtime.Scheme
	registry registry.Registry
}

// NewOperation creates a new internal installation operation object.
func NewOperation(log logr.Logger, c client.Client, scheme *runtime.Scheme, registry registry.Registry) Operation {
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

// Scheme returns a kubernetes scheme
func (o *operation) Scheme() *runtime.Scheme {
	return o.scheme
}

// Registry returns a registry.Registry instance
func (o *operation) Registry() registry.Registry {
	return o.registry
}

// UpdateInstallationStatus updates the status of a installation
func (o *operation) UpdateInstallationStatus(ctx context.Context, inst *lsv1alpha1.ComponentInstallation, phase lsv1alpha1.ComponentInstallationPhase, updatedConditions ...lsv1alpha1.Condition) error {
	inst.Status.Phase = phase
	inst.Status.Conditions = lsv1alpha1helper.MergeConditions(inst.Status.Conditions, updatedConditions...)
	inst.Status.ObservedGeneration = inst.Generation
	if err := o.client.Status().Update(ctx, inst); err != nil {
		o.log.Error(err, "unable to update installation status")
		return err
	}
	return nil
}
