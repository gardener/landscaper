// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lserrors "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

// executionItem is the internal representation of a execution item with its deployitem and status
type executionItem struct {
	Info       lsv1alpha1.DeployItemTemplate
	DeployItem *lsv1alpha1.DeployItem
}

// deployOrTrigger creates a new deployitem or triggers it if it already exists.
func (o *Operation) deployOrTrigger(ctx context.Context, item executionItem) lserrors.LsError {
	deployItemExists := item.DeployItem != nil

	if !deployItemExists {
		item.DeployItem = &lsv1alpha1.DeployItem{}
		item.DeployItem.GenerateName = fmt.Sprintf("%s-%s-", o.exec.Name, item.Info.Name)
		item.DeployItem.Namespace = o.exec.Namespace
	}
	item.DeployItem.Spec.RegistryPullSecrets = o.exec.Spec.RegistryPullSecrets

	if _, err := o.Writer().CreateOrUpdateDeployItem(ctx, read_write_layer.W000036, item.DeployItem, func() error {
		ApplyDeployItemTemplate(item.DeployItem, item.Info)
		kutil.SetMetaDataLabel(&item.DeployItem.ObjectMeta, lsv1alpha1.ExecutionManagedByLabel, o.exec.Name)
		item.DeployItem.Spec.Context = o.exec.Spec.Context
		o.Scheme().Default(item.DeployItem)
		return controllerutil.SetControllerReference(o.exec, item.DeployItem, o.Scheme())
	}); err != nil {
		msg := fmt.Sprintf("error while creating deployitem %q", item.Info.Name)
		if deployItemExists {
			msg = fmt.Sprintf("error while triggering deployitem %s", item.DeployItem.Name)
		}
		return lserrors.NewWrappedError(err, "TriggerDeployItem", msg, err.Error())
	}

	ref := lsv1alpha1.VersionedNamedObjectReference{}
	ref.Name = item.Info.Name
	ref.Reference.Name = item.DeployItem.Name
	ref.Reference.Namespace = item.DeployItem.Namespace
	ref.Reference.ObservedGeneration = item.DeployItem.Generation

	o.exec.Status.DeployItemReferences = lsv1alpha1helper.SetVersionedNamedObjectReference(o.exec.Status.DeployItemReferences, ref)
	o.exec.Status.ExecutionGenerations = setExecutionGeneration(o.exec.Status.ExecutionGenerations, item.Info.Name, o.exec.Generation)
	if err := o.Writer().UpdateExecutionStatus(ctx, read_write_layer.W000034, o.exec); err != nil {
		msg := fmt.Sprintf("unable to patch execution status %s", o.exec.Name)
		return lserrors.NewWrappedError(err, "TriggerDeployItem", msg, err.Error())
	}
	return nil
}

// CollectAndUpdateExportsNew loads all exports of all deployitems and persists them in a data object in the cluster.
// It also updates the export reference of the execution.
func (o *Operation) CollectAndUpdateExportsNew(ctx context.Context) lserrors.LsError {
	op := "CollectAndUpdateExports"

	items, _, lsErr := o.getDeployItems(ctx)
	if lsErr != nil {
		return lsErr
	}

	values := make(map[string]interface{})
	for _, item := range items {
		data, err := o.addExports(ctx, item.DeployItem)
		if err != nil {
			return lserrors.NewWrappedError(err, op, "AddExports", err.Error())
		}
		values[item.Info.Name] = data
	}

	if err := o.CreateOrUpdateExportReference(ctx, values); err != nil {
		return lserrors.NewWrappedError(err, op, "CreateOrUpdateExportReference", err.Error())
	}

	return nil
}

// addExports loads the exports of a deployitem and adds it to the given values.
func (o *Operation) addExports(ctx context.Context, item *lsv1alpha1.DeployItem) (map[string]interface{}, error) {
	if item.Status.ExportReference == nil {
		return nil, nil
	}
	secret := &corev1.Secret{}
	if err := o.Client().Get(ctx, item.Status.ExportReference.NamespacedName(), secret); err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := yaml.Unmarshal(secret.Data[lsv1alpha1.DataObjectSecretDataKey], &data); err != nil {
		return nil, err
	}
	return data, nil
}

func setExecutionGeneration(objects []lsv1alpha1.ExecutionGeneration, name string, gen int64) []lsv1alpha1.ExecutionGeneration {
	for i, ref := range objects {
		if ref.Name == name {
			objects[i].ObservedGeneration = gen
			return objects
		}
	}
	return append(objects, lsv1alpha1.ExecutionGeneration{Name: name, ObservedGeneration: gen})
}

func removeExecutionGeneration(objects []lsv1alpha1.ExecutionGeneration, name string) []lsv1alpha1.ExecutionGeneration {
	for i, ref := range objects {
		if ref.Name == name {
			return append(objects[:i], objects[i+1:]...)
		}
	}
	return objects
}
