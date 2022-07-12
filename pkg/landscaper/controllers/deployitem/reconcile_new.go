// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployitem

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lserrors "github.com/gardener/landscaper/apis/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

func (con *controller) reconcileNew(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := con.log.WithValues("resource", req.NamespacedName.String())
	logger.V(7).Info("reconcile")

	di := &lsv1alpha1.DeployItem{}
	if err := read_write_layer.GetDeployItem(ctx, con.c, req.NamespacedName, di); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if di.Status.JobID == di.Status.JobIDFinished {
		logger.V(7).Info("deploy item is finished, nothing to do")
		return reconcile.Result{}, nil
	}

	// check pickup timeout
	if !HasBeenPickedUp(di) {
		if con.pickupTimeout != 0 {
			logger.V(7).Info("check for pickup timeout")

			exceeded, requeue := con.isPickupTimeoutExceeded(logger, di)
			if exceeded {
				err := con.writePickupTimeoutExceeded(ctx, logger, di)
				// if there was a pickup timeout, no need to check for anything else
				return reconcile.Result{}, err
			}

			if requeue == nil {
				return reconcile.Result{}, nil
			}
			return reconcile.Result{RequeueAfter: *requeue}, nil
		}

		return reconcile.Result{}, nil
	}

	// check aborting timeout
	if con.abortingTimeout != 0 && metav1.HasAnnotation(di.ObjectMeta, string(lsv1alpha1helper.AbortTimestamp)) {
		logger.V(7).Info("check for aborting timeout")

		exceeded, requeue, err := con.isAbortingTimeoutExceeded(logger, di)
		if err != nil {
			return reconcile.Result{}, err
		}

		if exceeded {
			err = con.writeAbortingTimeoutExceeded(ctx, logger, di)
			// if there was an aborting timeout, no need to check for anything else
			return reconcile.Result{}, err
		}

		if requeue == nil {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{RequeueAfter: *requeue}, nil
	}

	// check progressing timeout
	// only do something if progressing timeout detection is neither deactivated on the deploy item,
	// nor defaulted by the deploy item and deactivated by default
	if !((di.Spec.Timeout != nil && di.Spec.Timeout.Duration == 0) || (di.Spec.Timeout == nil && con.defaultTimeout == 0)) {
		logger.V(7).Info("check for progressing timeout")

		exceeded, requeue, err := con.isProgressingTimeoutExceeded(logger, di)
		if err != nil {
			return reconcile.Result{}, err
		}

		if exceeded {
			err = con.writeProgressingTimeoutExceeded(ctx, logger, di)
			return reconcile.Result{}, err
		}

		if requeue == nil {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{RequeueAfter: *requeue}, nil
	}

	return reconcile.Result{}, nil
}

func (con *controller) isPickupTimeoutExceeded(log logr.Logger, di *lsv1alpha1.DeployItem) (bool, *time.Duration) {
	waitingForPickupDuration := time.Since(di.Status.JobIDGenerationTime.Time)
	if waitingForPickupDuration >= con.pickupTimeout {
		return true, nil
	}

	// deploy item neither picked up nor timed out
	// => requeue shortly after expected timeout
	requeue := con.pickupTimeout - waitingForPickupDuration + (5 * time.Second)
	return false, &requeue
}

func (con *controller) writePickupTimeoutExceeded(ctx context.Context, log logr.Logger, di *lsv1alpha1.DeployItem) error {
	// no deployer has picked up the deploy item within the timeframe
	// => pickup timeout
	logger := log.WithValues("operation", "PickupTimeoutExceeded")
	logger.V(5).Info("pickup timeout occurred")

	di.Status.DeployItemPhase = lsv1alpha1.DeployItemPhaseFailed
	di.Status.JobIDFinished = di.Status.JobID
	di.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
	di.Status.ObservedGeneration = di.Generation
	di.Status.LastError = lserrors.UpdatedError(di.Status.LastError,
		lsv1alpha1.PickupTimeoutOperation,
		lsv1alpha1.PickupTimeoutReason,
		fmt.Sprintf("no deployer has reconciled this deployitem within %d seconds", con.pickupTimeout/time.Second),
		lsv1alpha1.ErrorTimeout,
	)

	if err := con.Writer().UpdateDeployItemStatus(ctx, read_write_layer.W000110, di); err != nil {
		logger.Error(err, "unable to set deployitem status")
		return err
	}

	return nil
}

func (con *controller) isAbortingTimeoutExceeded(log logr.Logger, di *lsv1alpha1.DeployItem) (bool, *time.Duration, error) {
	ts, err := lsv1alpha1helper.GetTimestampAnnotation(di.ObjectMeta, lsv1alpha1helper.AbortTimestamp)
	if err != nil {
		return false, nil, fmt.Errorf("unable to parse abort timestamp annotation: %w", err)
	}

	waitingForAbortDuration := time.Since(ts)
	if waitingForAbortDuration >= con.abortingTimeout {
		return true, nil, nil
	}

	// deploy item neither aborted nor timed out
	// => requeue shortly after expected timeout
	requeue := con.abortingTimeout - waitingForAbortDuration + (5 * time.Second)
	return false, &requeue, nil
}

func (con *controller) writeAbortingTimeoutExceeded(ctx context.Context, log logr.Logger, di *lsv1alpha1.DeployItem) error {
	// deploy item has not been aborted within the timeframe
	// => aborting timeout
	logger := log.WithValues("operation", "AbortingTimeoutExceeded")
	logger.V(5).Info("aborting timeout occurred")

	lsv1alpha1helper.RemoveAbortOperationAndTimestamp(&di.ObjectMeta)

	di.Status.DeployItemPhase = lsv1alpha1.DeployItemPhaseFailed
	di.Status.JobIDFinished = di.Status.JobID
	di.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
	di.Status.ObservedGeneration = di.Generation
	di.Status.LastError = lserrors.UpdatedError(di.Status.LastError,
		lsv1alpha1.AbortingTimeoutOperation,
		lsv1alpha1.AbortingTimeoutReason,
		fmt.Sprintf("deployer has not aborted progressing this deploy item within %d seconds",
			con.abortingTimeout/time.Second),
		lsv1alpha1.ErrorTimeout)

	if err := con.Writer().UpdateDeployItemStatus(ctx, read_write_layer.W000111, di); err != nil {
		// we might need to expose this as event on the deploy item
		logger.Error(err, "unable to set deployitem status")
		return err
	}

	if err := con.Writer().UpdateDeployItem(ctx, read_write_layer.W000109, di); err != nil {
		logger.Error(err, "unable to update deploy item")
		return err
	}

	return nil
}

func (con *controller) isProgressingTimeoutExceeded(log logr.Logger, di *lsv1alpha1.DeployItem) (bool, *time.Duration, error) {
	logger := log.WithValues("operation", "DetectProgressingTimeouts")

	// no progressing timeout if timestamp is zero or deploy item is in a final phase
	if di.Status.LastReconcileTime.IsZero() {
		logger.V(7).Info("deploy item is reconciled for the first time, nothing to do")
		return false, nil, nil
	}

	var progressingTimeout time.Duration
	if di.Spec.Timeout == nil { // timeout not specified in deploy item, use global default
		progressingTimeout = con.defaultTimeout
	} else {
		progressingTimeout = di.Spec.Timeout.Duration
	}

	progressingDuration := time.Since(di.Status.LastReconcileTime.Time)
	if progressingDuration >= progressingTimeout {
		return true, nil, nil
	}

	// deploy item not yet timed out
	// => requeue shortly after expected timeout
	requeue := progressingTimeout - progressingDuration + (5 * time.Second)
	return false, &requeue, nil
}

func (con *controller) writeProgressingTimeoutExceeded(ctx context.Context, log logr.Logger, di *lsv1alpha1.DeployItem) error {
	// the deployer has not finished processing this deploy item within the timeframe
	// => abort it
	logger := log.WithValues("operation", "DetectProgressingTimeouts")
	logger.V(5).Info("deploy item timed out, setting abort operation annotation")

	lsv1alpha1helper.SetAbortOperationAndTimestamp(&di.ObjectMeta)

	if err := con.Writer().UpdateDeployItem(ctx, read_write_layer.W000108, di); err != nil {
		logger.Error(err, "unable to update deploy item")
		return err
	}

	return nil
}
