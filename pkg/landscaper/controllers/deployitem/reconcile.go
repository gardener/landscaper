// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployitem

import (
	"context"
	"fmt"
	"time"

	lsutil "github.com/gardener/landscaper/pkg/utils"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

func (con *controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := con.log.StartReconcile(req)
	ctx = logging.NewContext(ctx, logger)

	con.workerCounter.EnterWithLog(logger, 70)
	defer con.workerCounter.Exit()

	di := &lsv1alpha1.DeployItem{}
	if err := read_write_layer.GetDeployItem(ctx, con.c, req.NamespacedName, di); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if di.Status.GetJobID() == di.Status.JobIDFinished {
		logger.Debug("deploy item is finished, nothing to do")
		return reconcile.Result{}, nil
	}

	// check pickup timeout
	if !HasBeenPickedUp(di) {
		if con.pickupTimeout != 0 {
			logger.Debug("check for pickup timeout")

			exceeded, requeue := con.isPickupTimeoutExceeded(di)
			if exceeded {
				err := con.writePickupTimeoutExceeded(ctx, di)
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

	// check progressing timeout
	// only do something if progressing timeout detection is neither deactivated on the deploy item,
	// nor defaulted by the deploy item and deactivated by default
	if !((di.Spec.Timeout != nil && di.Spec.Timeout.Duration == 0) || (di.Spec.Timeout == nil && con.defaultTimeout == 0)) {
		logger.Debug("check for progressing timeout")

		var progressingTimeout time.Duration
		if di.Spec.Timeout == nil { // timeout not specified in deploy item, use global default
			progressingTimeout = con.defaultTimeout
		} else {
			progressingTimeout = di.Spec.Timeout.Duration
		}

		exceeded, requeue, err := con.isProgressingTimeoutExceeded(ctx, di, progressingTimeout)
		if err != nil {
			return reconcile.Result{}, err
		}

		if exceeded {
			err = con.writeProgressingTimeoutExceeded(ctx, di, progressingTimeout)
			return reconcile.Result{}, err
		}

		if requeue == nil {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{RequeueAfter: *requeue}, nil
	}

	return reconcile.Result{}, nil
}

func (con *controller) isPickupTimeoutExceeded(di *lsv1alpha1.DeployItem) (bool, *time.Duration) {
	waitingForPickupDuration := time.Duration(0)
	if di.Status.JobIDGenerationTime != nil {
		waitingForPickupDuration = time.Since(di.Status.JobIDGenerationTime.Time)
	}
	if waitingForPickupDuration >= con.pickupTimeout {
		return true, nil
	}

	// deploy item neither picked up nor timed out
	// => requeue shortly after expected timeout
	requeue := con.pickupTimeout - waitingForPickupDuration + (5 * time.Second)
	return false, &requeue
}

func (con *controller) writePickupTimeoutExceeded(ctx context.Context, di *lsv1alpha1.DeployItem) error {
	// no deployer has picked up the deploy item within the timeframe
	// => pickup timeout
	logger, ctx := logging.FromContextOrNew(ctx, nil)
	logger = logger.WithValues(lc.KeyMethod, "writePickupTimeoutExceeded")
	logger.Info("pickup timeout occurred")

	di.Status.JobIDFinished = di.Status.GetJobID()
	di.Status.ObservedGeneration = di.Generation
	lsv1alpha1helper.SetDeployItemToFailed(di)
	lsutil.SetLastError(&di.Status, lserrors.UpdatedError(di.Status.GetLastError(),
		lsv1alpha1.PickupTimeoutOperation,
		lsv1alpha1.PickupTimeoutReason,
		fmt.Sprintf("no deployer has reconciled this deployitem within %d seconds", con.pickupTimeout/time.Second),
		lsv1alpha1.ErrorTimeout,
	))

	if err := con.Writer().UpdateDeployItemStatus(ctx, read_write_layer.W000110, di); err != nil {
		logger.Error(err, "unable to set deployitem status")
		return err
	}

	return nil
}

func (con *controller) isProgressingTimeoutExceeded(ctx context.Context, di *lsv1alpha1.DeployItem, progressingTimeout time.Duration) (bool, *time.Duration, error) {
	logger, _ := logging.FromContextOrNew(ctx, nil)
	logger = logger.WithValues(lc.KeyMethod, "isProgressingTimeoutExceeded")

	if di.Status.LastReconcileTime.IsZero() {
		// should not happen
		logger.Debug("deploy item is reconciled for the first time, nothing to do")
		return false, nil, nil
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

func (con *controller) writeProgressingTimeoutExceeded(ctx context.Context, di *lsv1alpha1.DeployItem, progressingTimeout time.Duration) error {
	// the deployer has not finished processing this deploy item within the timeframe
	// => set to failed

	operation := "writeProgressingTimeoutExceeded"

	logger, ctx := logging.FromContextOrNew(ctx, nil)
	logger = logger.WithValues(lc.KeyMethod, operation)
	logger.Info("deploy item progressing timed out, setting to failed")

	di.Status.JobIDFinished = di.Status.GetJobID()
	di.Status.ObservedGeneration = di.Generation
	lsv1alpha1helper.SetDeployItemToFailed(di)
	lsutil.SetLastError(&di.Status, lserrors.UpdatedError(di.Status.GetLastError(),
		operation,
		lsv1alpha1.ProgressingTimeoutReason,
		fmt.Sprintf("deployer has not finished this deploy item within %d seconds", progressingTimeout/time.Second),
		lsv1alpha1.ErrorTimeout))

	if err := con.Writer().UpdateDeployItemStatus(ctx, read_write_layer.W000111, di); err != nil {
		// we might need to expose this as event on the deploy item
		logger.Error(err, "unable to set deployitem status")
		return err
	}

	return nil
}
