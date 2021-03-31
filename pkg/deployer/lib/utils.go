// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package lib

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
)

// HandleAnnotationsAndGeneration is meant to be called at the beginning of a deployer's reconcile loop.
// If a reconcile is needed due to the reconcile annotation or a change in the generation, it will set the phase to Init and remove the reconcile annotation.
// It will also remove the timeout annotation if it is set.
// Returns:
//   - the modified deployitem
//   - an error, if updating the deployitem failed, nil otherwise
func HandleAnnotationsAndGeneration(ctx context.Context, log logr.Logger, c client.Client, di *lsv1alpha1.DeployItem) error {
	changedMeta := false
	hasReconcileAnnotation := lsv1alpha1helper.HasOperation(di.ObjectMeta, lsv1alpha1.ReconcileOperation)
	if hasReconcileAnnotation || di.Status.ObservedGeneration != di.Generation {
		// reconcile necessary due to one of
		// - reconcile annotation
		// - outdated generation
		log.V(5).Info("reconcile required, setting observed generation, phase, and last change reconcile timestamp", "reconcileAnnotation", hasReconcileAnnotation, "observedGeneration", di.Status.ObservedGeneration, "generation", di.Generation)
		di.Status.ObservedGeneration = di.Generation
		di.Status.Phase = lsv1alpha1.ExecutionPhaseInit
		now := metav1.Now()
		di.Status.LastChangeReconcileTime = &now

		log.V(7).Info("updating status")
		if err := c.Status().Update(ctx, di); err != nil {
			return err
		}
		log.V(7).Info("successfully updated status")
	}
	if hasReconcileAnnotation {
		log.V(5).Info("removing reconcile annotation")
		changedMeta = true
		delete(di.ObjectMeta.Annotations, lsv1alpha1.OperationAnnotation)
	}
	if lsv1alpha1helper.HasTimestampAnnotation(di.ObjectMeta, lsv1alpha1helper.ReconcileTimestamp) {
		log.V(5).Info("removing timestamp annotation")
		changedMeta = true
		delete(di.ObjectMeta.Annotations, lsv1alpha1.ReconcileTimestampAnnotation)
	}

	if changedMeta {
		log.V(7).Info("updating metadata")
		if err := c.Update(ctx, di); err != nil {
			return err
		}
		log.V(7).Info("successfully updated metadata")
	}

	return nil
}

// ShouldReconcile returns true if the given deploy item should be reconciled
func ShouldReconcile(di *lsv1alpha1.DeployItem) bool {
	if di.Status.Phase == lsv1alpha1.ExecutionPhaseInit || di.Status.Phase == lsv1alpha1.ExecutionPhaseProgressing || di.Status.Phase == lsv1alpha1.ExecutionPhaseDeleting {
		return true
	}

	return false
}
