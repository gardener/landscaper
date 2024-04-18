// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"context"
	"fmt"

	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lserrors "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/landscaper/targetresolver/secret"
	"github.com/gardener/landscaper/pkg/utils/clusters"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

const clusterNameAnnotation = "landscaper.gardener.cloud/clustername"

// executionItem is the internal representation of a execution item with its deployitem and status
type executionItem struct {
	Info       lsv1alpha1.DeployItemTemplate
	DeployItem *lsv1alpha1.DeployItem
}

// deployOrTrigger creates a new deployitem or triggers it if it already exists.
func (o *Operation) updateDeployItem(ctx context.Context, item executionItem) (*lsv1alpha1.DiNamePair, lserrors.LsError) {
	op := "updateDeployItem"

	clusterName, err := o.getShootClusterName(ctx, item.Info)
	if err != nil {
		return nil, err
	}

	deployItemExists := item.DeployItem != nil

	if !deployItemExists {
		item.DeployItem = &lsv1alpha1.DeployItem{}
		item.DeployItem.GenerateName = fmt.Sprintf("%s-%s-", o.exec.Name, item.Info.Name)
		item.DeployItem.Namespace = o.exec.Namespace
	}

	if _, err := o.WriterToLsUncachedClient().CreateOrUpdateDeployItem(ctx, read_write_layer.W000036, item.DeployItem, func() error {
		if item.DeployItem.CreationTimestamp.IsZero() && item.DeployItem.DeletionTimestamp.IsZero() {
			controllerutil.AddFinalizer(item.DeployItem, lsv1alpha1.LandscaperFinalizer)
		}
		ApplyDeployItemTemplate(item.DeployItem, item.Info)
		kutil.SetMetaDataLabel(&item.DeployItem.ObjectMeta, lsv1alpha1.ExecutionManagedByLabel, o.exec.Name)
		item.DeployItem.Spec.Context = o.exec.Spec.Context
		if len(clusterName) > 0 {
			metav1.SetMetaDataAnnotation(&item.DeployItem.ObjectMeta, clusterNameAnnotation, clusterName)
		}

		lsv1alpha1helper.DeleteCacheHelmChartsAnnotation(&item.DeployItem.ObjectMeta)
		if lsv1alpha1helper.HasCacheHelmChartsAnnotation(&o.exec.ObjectMeta) {
			metav1.SetMetaDataAnnotation(&item.DeployItem.ObjectMeta, lsv1alpha1.CacheHelmChartsAnnotation, "true")
		}

		o.Scheme().Default(item.DeployItem)
		return controllerutil.SetControllerReference(o.exec, item.DeployItem, o.Scheme())
	}); err != nil {
		msg := fmt.Sprintf("error while creating deployitem %q", item.Info.Name)
		if deployItemExists {
			msg = fmt.Sprintf("error while triggering deployitem %s", item.DeployItem.Name)
		}
		return nil, lserrors.NewWrappedError(err, op, msg, err.Error())
	}

	return &lsv1alpha1.DiNamePair{
		SpecName:   item.Info.Name,
		ObjectName: item.DeployItem.Name,
	}, nil
}

// getShootClusterName determines for a deployitem whether the "skipUninstallIfClusterRemoved" feature is enabled,
// and whether its target is managed by the "targetsync" mechanism. In this case, the name of the Gardener shoot cluster
// is returned, otherwise an empty string. (For the "skipUninstallIfClusterRemoved" feature, a deployitem is
// annotated with the shoot cluster name at creation/update time, so that later at deletion time the existence of the
// shoot cluster can be checked. The existence check of the shoot resource uses the same client as the targetsync.)
func (o *Operation) getShootClusterName(ctx context.Context, info lsv1alpha1.DeployItemTemplate) (string, lserrors.LsError) {
	op := "getShootClusterName"

	if info.OnDelete == nil || !info.OnDelete.SkipUninstallIfClusterRemoved || info.Target == nil {
		return "", nil
	}

	target := &lsv1alpha1.Target{}
	targetKey := client.ObjectKey{Namespace: o.exec.Namespace, Name: info.Target.Name}
	if err := o.LsUncachedClient().Get(ctx, targetKey, target); err != nil {
		msg := fmt.Sprintf("unable to fetch target %s/%s", o.exec.Namespace, info.Target.Name)
		return "", lserrors.NewWrappedError(err, op, msg, err.Error())
	}

	if !clusters.HasTargetSyncLabel(target) {
		return "", nil
	}

	targetResolver := secret.New(o.LsUncachedClient())
	kubeconfigBytes, err := targetResolver.GetKubeconfigFromTarget(ctx, target)
	if err != nil {
		msg := fmt.Sprintf("unable to retrieve kubeconfig from target %s/%s", o.exec.Namespace, info.Target.Name)
		return "", lserrors.NewWrappedError(err, op, msg, err.Error())
	}

	clusterName, err := clusters.GetShootClusterNameFromKubeconfig(ctx, kubeconfigBytes)
	if err != nil {
		msg := fmt.Sprintf("unable to retrieve shoot cluster name from target %s/%s", o.exec.Namespace, info.Target.Name)
		return "", lserrors.NewWrappedError(err, op, msg, err.Error())
	}

	return clusterName, nil
}

// CollectAndUpdateExportsNew loads all exports of all deployitems and persists them in a data object in the cluster.
// It also updates the export reference of the execution.
func (o *Operation) CollectAndUpdateExportsNew(ctx context.Context) lserrors.LsError {
	op := "CollectAndUpdateExports"

	items, _, lsErr := o.getDeployItems(ctx, o.exec.Status.DeployItemCache)
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
	if err := o.LsUncachedClient().Get(ctx, item.Status.ExportReference.NamespacedName(), secret); err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := yaml.Unmarshal(secret.Data[lsv1alpha1.DataObjectSecretDataKey], &data); err != nil {
		return nil, err
	}
	return data, nil
}
