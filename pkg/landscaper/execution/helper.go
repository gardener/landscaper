// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// ApplyDeployItemTemplate sets and updates the values defined by deploy item template on a deploy item.
func ApplyDeployItemTemplate(di *lsv1alpha1.DeployItem, tmpl lsv1alpha1.DeployItemTemplate) {
	lsv1alpha1helper.SetOperation(&di.ObjectMeta, lsv1alpha1.ReconcileOperation)
	di.Spec.Type = tmpl.Type
	di.Spec.Target = tmpl.Target
	di.Spec.Configuration = tmpl.Configuration
	for k, v := range tmpl.Labels {
		kutil.SetMetaDataLabel(&di.ObjectMeta, k, v)
	}
	kutil.SetMetaDataLabel(&di.ObjectMeta, lsv1alpha1.ExecutionManagedNameLabel, tmpl.Name)
	metav1.SetMetaDataAnnotation(&di.ObjectMeta, lsv1alpha1.ExecutionDependsOnAnnotation, strings.Join(tmpl.DependsOn, ","))
}

func getDeployItemIndexByManagedName(items []lsv1alpha1.DeployItem, name string) (int, bool) {
	for i, item := range items {
		if ann := item.Labels[lsv1alpha1.ExecutionManagedNameLabel]; ann == name {
			return i, true
		}
	}

	return -1, false
}

// listManagedDeployItems collects all deploy items that are managed by the execution.
// The managed execution is identified by the managed by label and the ownership.
func (o *Operation) listManagedDeployItems(ctx context.Context) ([]lsv1alpha1.DeployItem, error) {
	deployItemList := &lsv1alpha1.DeployItemList{}
	// todo: maybe use name and namespace
	if err := o.Client().List(ctx, deployItemList, client.MatchingLabels{lsv1alpha1.ExecutionManagedByLabel: o.exec.Name}, client.InNamespace(o.exec.Namespace)); err != nil {
		return nil, err
	}
	return deployItemList.Items, nil
}

// getExecutionItems creates an internal representation for all execution items.
// It also returns all removed deploy items that are not defined by the execution anymore.
func (o *Operation) getExecutionItems(items []lsv1alpha1.DeployItem) ([]executionItem, []lsv1alpha1.DeployItem) {
	execItems := make([]executionItem, len(o.exec.Spec.DeployItems))
	managed := sets.NewInt()
	for i, exec := range o.exec.Spec.DeployItems {
		execItem := executionItem{
			Info: *exec.DeepCopy(),
		}
		if j, found := getDeployItemIndexByManagedName(items, exec.Name); found {
			managed.Insert(j)
			execItem.DeployItem = items[j].DeepCopy()
		}
		execItems[i] = execItem
	}
	orphaned := make([]lsv1alpha1.DeployItem, 0)
	for i, item := range items {
		if !managed.Has(i) {
			orphaned = append(orphaned, item)
		}
	}
	return execItems, orphaned
}
