// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"context"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

// ApplyDeployItemTemplate sets and updates the values defined by deploy item template on a deploy item.
func ApplyDeployItemTemplate(di *lsv1alpha1.DeployItem, tmpl lsv1alpha1.DeployItemTemplate) {
	lsv1alpha1helper.SetTimestampAnnotationNow(&di.ObjectMeta, lsv1alpha1helper.ReconcileTimestamp)
	di.Spec.Type = tmpl.Type
	di.Spec.Target = tmpl.Target
	di.Spec.Configuration = tmpl.Configuration
	di.Spec.Timeout = tmpl.Timeout
	di.Spec.UpdateOnChangeOnly = tmpl.UpdateOnChangeOnly
	di.Spec.OnDelete = tmpl.OnDelete
	for k, v := range tmpl.Labels {
		kutil.SetMetaDataLabel(&di.ObjectMeta, k, v)
	}
	kutil.SetMetaDataLabel(&di.ObjectMeta, lsv1alpha1.ExecutionManagedNameLabel, tmpl.Name)
	metav1.SetMetaDataAnnotation(&di.ObjectMeta, lsv1alpha1.ExecutionDependsOnAnnotation, strings.Join(tmpl.DependsOn, ","))
	metav1.SetMetaDataAnnotation(&di.ObjectMeta, lsv1alpha1.DeployerTypeAnnotation, string(tmpl.Type))
	targetName := lsv1alpha1.NoTargetNameValue
	if di.Spec.Target != nil && di.Spec.Target.Name != "" {
		targetName = di.Spec.Target.Name
	}
	metav1.SetMetaDataAnnotation(&di.ObjectMeta, lsv1alpha1.DeployerTargetNameAnnotation, targetName)
}

func getDeployItemIndexByManagedName(items []*lsv1alpha1.DeployItem, name string) (int, bool) {
	for i, item := range items {
		if ann := item.Labels[lsv1alpha1.ExecutionManagedNameLabel]; ann == name {
			return i, true
		}
	}

	return -1, false
}

// listManagedDeployItems collects all deploy items that are managed by the execution.
// The managed execution is identified by the managed by label and the ownership.
func (o *Operation) ListManagedDeployItems(ctx context.Context, readID read_write_layer.ReadID,
	deployItemCache *lsv1alpha1.DeployItemCache) ([]*lsv1alpha1.DeployItem, error) {

	deployItems := []*lsv1alpha1.DeployItem{}

	if deployItemCache != nil {
		for i := range deployItemCache.OrphanedDIs {
			nextDi := &lsv1alpha1.DeployItem{}
			key := client.ObjectKey{Namespace: o.exec.Namespace, Name: deployItemCache.OrphanedDIs[i]}
			if err := read_write_layer.GetDeployItem(ctx, o.Client(), key, nextDi, readID); err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				return nil, err
			}
			deployItems = append(deployItems, nextDi)
		}

		for i := range deployItemCache.ActiveDIs {
			nextDi := &lsv1alpha1.DeployItem{}
			key := client.ObjectKey{Namespace: o.exec.Namespace, Name: deployItemCache.ActiveDIs[i].ObjectName}
			if err := read_write_layer.GetDeployItem(ctx, o.Client(), key, nextDi, readID); err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				return nil, err
			}
			deployItems = append(deployItems, nextDi)
		}

		return deployItems, nil

	} else {
		deployItemList, err := read_write_layer.ListManagedDeployItems(ctx, o.Client(), client.ObjectKeyFromObject(o.exec), readID)
		if err != nil {
			return nil, err
		}

		for i := range deployItemList.Items {
			deployItems = append(deployItems, &deployItemList.Items[i])
		}
		return deployItems, nil
	}
}

// getExecutionItems creates an internal representation for all execution items.
// It also returns all removed deploy items that are not defined by the execution anymore.
func (o *Operation) getExecutionItems(items []*lsv1alpha1.DeployItem) ([]*executionItem, []*lsv1alpha1.DeployItem) {
	execItems := make([]*executionItem, len(o.exec.Spec.DeployItems))
	managed := sets.NewInt()
	for i, di := range o.exec.Spec.DeployItems {
		execItem := executionItem{
			Info: *di.DeepCopy(),
		}
		if j, found := getDeployItemIndexByManagedName(items, di.Name); found {
			managed.Insert(j)
			execItem.DeployItem = items[j].DeepCopy()
		}
		execItems[i] = &execItem
	}
	orphaned := make([]*lsv1alpha1.DeployItem, 0)
	for i, item := range items {
		if !managed.Has(i) {
			orphaned = append(orphaned, item)
		}
	}
	return execItems, orphaned
}
