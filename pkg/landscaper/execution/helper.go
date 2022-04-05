// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"context"
	"fmt"
	"strings"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
)

// ApplyDeployItemTemplate sets and updates the values defined by deploy item template on a deploy item.
func ApplyDeployItemTemplate(di *lsv1alpha1.DeployItem, tmpl lsv1alpha1.DeployItemTemplate) {
	lsv1alpha1helper.SetOperation(&di.ObjectMeta, lsv1alpha1.ReconcileOperation)
	lsv1alpha1helper.SetTimestampAnnotationNow(&di.ObjectMeta, lsv1alpha1helper.ReconcileTimestamp)
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
	if err := read_write_layer.ListDeployItems(ctx, o.Client(), deployItemList,
		client.MatchingLabels{lsv1alpha1.ExecutionManagedByLabel: o.exec.Name}, client.InNamespace(o.exec.Namespace)); err != nil {
		return nil, err
	}
	return deployItemList.Items, nil
}

// getExecutionItems creates an internal representation for all execution items.
// It also returns all removed deploy items that are not defined by the execution anymore.
func (o *Operation) getExecutionItems(items []lsv1alpha1.DeployItem) ([]executionItem, []lsv1alpha1.DeployItem) {
	execItems := make([]executionItem, len(o.exec.Spec.DeployItems))
	managed := sets.NewInt()
	for i, di := range o.exec.Spec.DeployItems {
		execItem := executionItem{
			Info: *di.DeepCopy(),
		}
		if j, found := getDeployItemIndexByManagedName(items, di.Name); found {
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

// HandleDeployItemPhaseAndGenerationChanges updates the phase of the given execution, if its phase doesn't match the combined phase of its deploy items anymore.
// If a deploy item's generation differs from the observed one or a deploy item doesn't exist, the phase is set to 'Progressing'.
// If the phase is changed to 'Succeeded', the exports of the deploy items are updated.
func (o *Operation) HandleDeployItemPhaseAndGenerationChanges(ctx context.Context, logger logr.Logger) error {
	deployitems := []lsv1alpha1.DeployItem{}
	phases := []lsv1alpha1.ExecutionPhase{}
	// fetch all managed deploy items and check their phase and generation
	for _, managedDi := range o.exec.Status.DeployItemReferences {
		di := &lsv1alpha1.DeployItem{}
		err := read_write_layer.GetDeployItem(ctx, o.Client(), managedDi.Reference.NamespacedName(), di)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("unable to get deploy item %q: %w", managedDi.Reference.NamespacedName().String(), err)
			} else {
				di = nil
			}
		}

		if di == nil || managedDi.Reference.ObservedGeneration != di.Generation {
			// at least one deploy item is outdated or got deleted, a reconcile is required
			logger.V(7).Info("deploy item cannot be found or does not match last observed generation")
			o.exec.Status.Phase = lsv1alpha1.ExecutionPhaseProgressing
			err := o.Writer().UpdateExecutionStatus(ctx, read_write_layer.W000028, o.exec)
			if err != nil {
				return fmt.Errorf("error updating execution status for %s/%s: %w", o.exec.Namespace, o.exec.Name, err)
			}
			return nil
		}
		phases = append(phases, di.Status.Phase)
		deployitems = append(deployitems, *di)
	}

	cp := lsv1alpha1helper.CombinedExecutionPhase(phases...)
	if o.exec.Status.Phase != cp {
		// Phase is completed but doesn't fit to the deploy items' phases
		logger.V(5).Info("execution phase mismatch", "phase", string(o.exec.Status.Phase), "combinedPhase", string(cp))
		o.exec.Status.Phase = cp
		err := o.Writer().UpdateExecutionStatus(ctx, read_write_layer.W000030, o.exec)
		if err != nil {
			return fmt.Errorf("error updating execution status for %s/%s: %w", o.exec.Namespace, o.exec.Name, err)
		}

		if cp == lsv1alpha1.ExecutionPhaseSucceeded {
			// phase changed to Succeeded, it might be necessary to generate the exports again
			logger.V(5).Info("phase changed to %q, compute deploy item exports again", string(lsv1alpha1.ExecutionPhaseSucceeded))
			execItems, _ := o.getExecutionItems(deployitems)
			err = o.collectAndUpdateExports(ctx, execItems)
			if err != nil {
				return fmt.Errorf("error while updating exports of execution %s/%s: %w", o.exec.Namespace, o.exec.Name, err)
			}
		}

		return nil
	}
	logger.V(7).Info("execution is in a final state and deployitems are up-to-date")

	return nil
}
