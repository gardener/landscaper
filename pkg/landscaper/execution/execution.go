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

package execution

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
)

// Operation contains all execution operations
type Operation struct {
	operation.Interface
	exec *lsv1alpha1.Execution
}

// NewOperation creates a new execution operations
func NewOperation(op operation.Interface, exec *lsv1alpha1.Execution) *Operation {
	return &Operation{
		Interface: op,
		exec:      exec,
	}
}

// UpdateStatus updates the status of a execution
func (o *Operation) UpdateStatus(ctx context.Context, phase lsv1alpha1.ExecutionPhase, updatedConditions ...lsv1alpha1.Condition) error {
	o.exec.Status.Phase = phase
	o.exec.Status.Conditions = lsv1alpha1helper.MergeConditions(o.exec.Status.Conditions, updatedConditions...)
	if err := o.Client().Status().Update(ctx, o.exec); err != nil {
		o.Log().Error(err, "unable to set installation status")
		return err
	}
	return nil
}

// CreateOrUpdateDataObject creates or updates a dataobject from a object reference
// if
func (o *Operation) CreateOrUpdateExportReference(ctx context.Context, values interface{}) error {
	obj := &corev1.Secret{}
	obj.GenerateName = "dataobject-"
	obj.Namespace = o.exec.Namespace
	if o.exec.Status.ExportReference != nil {
		obj.Name = o.exec.Status.ExportReference.Name
		obj.Namespace = o.exec.Status.ExportReference.Namespace
	}
	data, err := yaml.Marshal(values)
	if err != nil {
		return err
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, o.Client(), obj, func() error {
		obj.Data = map[string][]byte{
			lsv1alpha1.DataObjectSecretDataKey: data,
		}
		return nil
	}); err != nil {
		return err
	}

	o.exec.Status.ExportReference = &lsv1alpha1.ObjectReference{
		Name:      obj.Name,
		Namespace: obj.Namespace,
	}
	return o.UpdateStatus(ctx, o.exec.Status.Phase)
}
