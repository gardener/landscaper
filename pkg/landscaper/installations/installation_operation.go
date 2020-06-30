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
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// InstallationOperation contains all installation operations
type InstallationOperation struct {
	Operation

	Inst    *Installation
	context *Context
}

// NewInstallationOperation creates a new installation operation
func NewInstallationOperation(ctx context.Context, op Operation, inst *Installation) (*InstallationOperation, error) {
	var err error
	instOp := &InstallationOperation{
		Operation: op,
		Inst:      inst,
	}

	instOp.context, err = instOp.DetermineContext(ctx)
	if err != nil {
		return nil, err
	}

	return instOp, nil
}

// Context returns the context of the operated installation
func (o *InstallationOperation) Context() *Context {
	return o.context
}

// GetRootInstallations returns all root installations in the system
func (o *InstallationOperation) GetRootInstallations(ctx context.Context, opts ...client.ListOption) ([]*lsv1alpha1.ComponentInstallation, error) {
	r, err := labels.NewRequirement(lsv1alpha1.EncompassedByLabel, selection.DoesNotExist, nil)
	if err != nil {
		return nil, err
	}
	opts = append(opts, client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(*r)})

	installationList := &lsv1alpha1.ComponentInstallationList{}
	if err := o.Client().List(ctx, installationList, opts...); err != nil {
		return nil, err
	}

	installations := make([]*lsv1alpha1.ComponentInstallation, len(installationList.Items))
	for i, obj := range installationList.Items {
		inst := obj
		installations[i] = &inst
	}
	return installations, nil
}

// TriggerDependants triggers all installations that depend on the current installation.
// These are most likely all installation that import a key which is exported by the current installation.
func (o *InstallationOperation) TriggerDependants(ctx context.Context) error {

	for _, sibling := range o.Context().Siblings {
		if !importsAnyExport(o.Inst, sibling) {
			continue
		}

		// todo: maybe use patch
		metav1.SetMetaDataAnnotation(&sibling.Info.ObjectMeta, lsv1alpha1.OperationAnnotation, string(lsv1alpha1.ReconcileOperation))
		if err := o.Client().Update(ctx, sibling.Info); err != nil {
			return errors.Wrapf(err, "unable to trigger installation %s", sibling.Info.Name)
		}
	}

	return nil
}

// UpdateExportReference updates the data object that holds the exported values of the installation.
func (o *InstallationOperation) UpdateExportReference(ctx context.Context, values interface{}) error {
	obj := &corev1.Secret{}
	obj.Name = fmt.Sprintf("%s-imports", o.Inst.Info.Name)
	obj.Namespace = o.Inst.Info.Namespace
	if o.Inst.Info.Status.ExportReference != nil {
		obj.Name = o.Inst.Info.Status.ExportReference.Name
		obj.Namespace = o.Inst.Info.Status.ExportReference.Namespace
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

	o.Inst.Info.Status.ExportReference = &lsv1alpha1.ObjectReference{
		Name:      obj.Name,
		Namespace: obj.Namespace,
	}
	return o.UpdateInstallationStatus(ctx, o.Inst.Info, o.Inst.Info.Status.Phase)
}

// UpdateImportReference updates the data object that holds the imported values
// todo: make general import/export dataobject creation and update
func (o *InstallationOperation) UpdateImportReference(ctx context.Context, values interface{}) error {
	obj := &corev1.Secret{}
	obj.Name = fmt.Sprintf("%s-imports", o.Inst.Info.Name)
	obj.Namespace = o.Inst.Info.Namespace
	if o.Inst.Info.Status.ImportReference != nil {
		obj.Name = o.Inst.Info.Status.ImportReference.Name
		obj.Namespace = o.Inst.Info.Status.ImportReference.Namespace
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

	o.Inst.Info.Status.ImportReference = &lsv1alpha1.ObjectReference{
		Name:      obj.Name,
		Namespace: obj.Namespace,
	}
	return o.UpdateInstallationStatus(ctx, o.Inst.Info, o.Inst.Info.Status.Phase)
}

func importsAnyExport(exporter, importer *Installation) bool {
	for _, export := range exporter.Info.Spec.Exports {
		if _, err := importer.GetImportMappingFrom(export.To); err != nil {
			return true
		}
	}
	return false
}
