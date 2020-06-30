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
	"github.com/gardener/landscaper/pkg/landscaper/datatype"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/landscaper/registry"
)

// Operation is the operation interface that is used to share common operational data across the installation reconciler.
type Operation interface {
	lsoperation.Interface
	GetDataType(name string) (*datatype.Datatype, bool)
	UpdateInstallationStatus(ctx context.Context, inst *lsv1alpha1.ComponentInstallation, phase lsv1alpha1.ComponentInstallationPhase, updatedConditions ...lsv1alpha1.Condition) error
}

type operation struct {
	lsoperation.Interface

	datatypes map[string]*datatype.Datatype
}

// NewOperation creates a new internal installation operation object.
func NewOperation(log logr.Logger, c client.Client, scheme *runtime.Scheme, registry registry.Registry, datatypes map[string]*datatype.Datatype) Operation {
	return &operation{
		Interface: lsoperation.NewOperation(log, c, scheme, registry),
		datatypes: datatypes,
	}
}

// GetDataType returns the datatype with a specific name.
// It returns ok = false if the datatype does not exist.
func (o *operation) GetDataType(name string) (dt *datatype.Datatype, ok bool) {
	dt, ok = o.datatypes[name]
	return
}

// UpdateInstallationStatus updates the status of a installation
func (o *operation) UpdateInstallationStatus(ctx context.Context, inst *lsv1alpha1.ComponentInstallation, phase lsv1alpha1.ComponentInstallationPhase, updatedConditions ...lsv1alpha1.Condition) error {
	inst.Status.Phase = phase
	inst.Status.Conditions = lsv1alpha1helper.MergeConditions(inst.Status.Conditions, updatedConditions...)
	if err := o.Client().Status().Update(ctx, inst); err != nil {
		o.Log().Error(err, "unable to set installation status")
		return err
	}
	return nil
}
