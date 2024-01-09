// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployitem

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	lsutil "github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

func (con *controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := con.log.StartReconcile(req)
	ctx = logging.NewContext(ctx, logger)

	con.workerCounter.EnterWithLog(logger, 70, "di-timeout")
	defer con.workerCounter.Exit()

	di := &lsv1alpha1.DeployItem{}
	if err := read_write_layer.GetDeployItem(ctx, con.lsUncachedClient, req.NamespacedName, di, read_write_layer.R000028); err != nil {
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

	if HasBeenPickedUp(di) || con.pickupTimeout == 0 {
		// deploy item has been picked up, or the pickup check is deactivated
		return reconcile.Result{}, nil
	}

	logger.Debug("check for pickup timeout")
	exceeded, requeue := con.isPickupTimeoutExceeded(di)
	if exceeded {
		// pickup timeout is exceeded

		// check if the reason is that the target does not exist
		targetNotFound := false
		if di.Spec.Target != nil && di.Spec.Target.Name != "" {
			target := &lsv1alpha1.Target{}
			target.SetName(di.Spec.Target.Name)
			target.SetNamespace(di.Namespace)
			if err := con.lsUncachedClient.Get(ctx, client.ObjectKeyFromObject(target), target); err != nil {
				if apierrors.IsNotFound(err) {
					targetNotFound = true
				}
			}
		}

		err := con.writePickupTimeoutExceeded(ctx, di, targetNotFound)
		return reconcile.Result{}, err
	}

	if requeue != nil {
		// pickup timeout not yet exceeded; check again at the time when it would be exceeded
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

func (con *controller) writePickupTimeoutExceeded(ctx context.Context, di *lsv1alpha1.DeployItem, reasonTargetNotFound bool) error {
	// no deployer has picked up the deploy item within the timeframe
	// => pickup timeout
	logger, ctx := logging.FromContextOrNew(ctx, nil)
	logger = logger.WithValues(lc.KeyMethod, "writePickupTimeoutExceeded")
	logger.Info("pickup timeout occurred", "reasonTargetNotFound", reasonTargetNotFound)

	di.Status.JobIDFinished = di.Status.GetJobID()
	di.Status.TransitionTimes = lsutil.SetFinishedTransitionTime(di.Status.TransitionTimes)
	di.Status.ObservedGeneration = di.Generation
	lsv1alpha1helper.SetDeployItemToFailed(di)
	targetReasonMsg := ""
	if reasonTargetNotFound {
		targetReasonMsg = ", probably because the referenced Target does not exist"
	}
	lsutil.SetLastError(&di.Status, lserrors.UpdatedError(di.Status.GetLastError(),
		lsv1alpha1.PickupTimeoutOperation,
		lsv1alpha1.PickupTimeoutReason,
		fmt.Sprintf("no deployer has reconciled this deployitem within %d seconds%s", con.pickupTimeout/time.Second, targetReasonMsg),
		lsv1alpha1.ErrorTimeout,
	))

	if err := con.Writer().UpdateDeployItemStatus(ctx, read_write_layer.W000110, di); err != nil {
		logger.Error(err, "unable to set deployitem status")
		return err
	}

	return nil
}
