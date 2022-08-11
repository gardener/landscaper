// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployitem

import (
	"context"
	"fmt"
	"reflect"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lscore "github.com/gardener/landscaper/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

// NewController creates a new deploy item controller that handles timeouts
// To detect pickup timeouts (when a DeployItem resource is not reconciled by any deployer within a specified timeframe), the controller checks for a timestamp annotation.
// It is expected that deployers remove the timestamp annotation from deploy items during reconciliation. If the timestamp annotation exists and is older than a specified duration,
// the controller marks the deploy item as failed.
// pickupTimeout is a string containing the pickup timeout duration, either as 'none' or as a duration that can be parsed by time.ParseDuration.
func NewController(logger logging.Logger, c client.Client, scheme *runtime.Scheme, pickupTimeout, abortingTimeout, defaultTimeout *lscore.Duration) (reconcile.Reconciler, error) {
	con := controller{log: logger, c: c, scheme: scheme}
	if pickupTimeout != nil {
		con.pickupTimeout = pickupTimeout.Duration
	} else {
		con.pickupTimeout = time.Duration(0)
	}
	if abortingTimeout != nil {
		con.abortingTimeout = abortingTimeout.Duration
	} else {
		con.abortingTimeout = time.Duration(0)
	}
	if defaultTimeout != nil {
		con.defaultTimeout = defaultTimeout.Duration
	} else {
		con.defaultTimeout = time.Duration(0)
	}

	// log pickup timeout
	logger.Info("deploy item pickup timeout detection", "active", con.pickupTimeout != 0, "timeout", con.pickupTimeout.String())
	logger.Info("deploy item aborting timeout detection", "active", con.abortingTimeout != 0, "timeout", con.abortingTimeout.String())
	logger.Info("deploy item default timeout", "active", con.defaultTimeout != 0, "timeout", con.defaultTimeout.String())

	return &con, nil
}

type controller struct {
	log             logging.Logger
	c               client.Client
	scheme          *runtime.Scheme
	pickupTimeout   time.Duration
	abortingTimeout time.Duration
	defaultTimeout  time.Duration
}

func (con *controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	if utils.IsNewReconcile() {
		return con.reconcileNew(ctx, req)
	} else {
		return con.reconcileOld(ctx, req)
	}
}

func (con *controller) reconcileOld(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := con.log.StartReconcile(req)
	ctx = logging.NewContext(ctx, logger)

	di := &lsv1alpha1.DeployItem{}
	if err := read_write_layer.GetDeployItem(ctx, con.c, req.NamespacedName, di); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	var requeue *time.Duration
	var err error
	old := di.DeepCopy()

	// detect pickup timeout
	if con.pickupTimeout != 0 {
		logger.Debug("check for pickup timeout")
		requeue, err = con.detectPickupTimeouts(ctx, di)
		if err != nil {
			return reconcile.Result{}, err
		}
		if !reflect.DeepEqual(old.Status, di.Status) {
			if err := con.Writer().UpdateDeployItemStatus(ctx, read_write_layer.W000056, di); err != nil {
				logger.Error(err, "unable to set deployitem status")
				return reconcile.Result{}, err
			}
			// if there was a pickup timeout, no need to check for anything else
			return reconcile.Result{}, nil
		}
	}

	// detect aborting timeout
	if con.abortingTimeout != 0 {
		logger.Debug("check for aborting timeout")
		tmp, err := con.detectAbortingTimeouts(ctx, di)
		if err != nil {
			return reconcile.Result{}, err
		}
		if requeue == nil {
			requeue = tmp
		} else if tmp != nil && *tmp < *requeue {
			requeue = tmp
		}
		if !reflect.DeepEqual(old.Status, di.Status) {
			if err := con.Writer().UpdateDeployItemStatus(ctx, read_write_layer.W000057, di); err != nil {
				// we might need to expose this as event on the deploy item
				logger.Error(err, "unable to set deployitem status")
				return reconcile.Result{}, err
			}
			// if there was an aborting timeout, no need to check for anything else
			return reconcile.Result{}, nil
		}
		if !reflect.DeepEqual(old.Annotations, di.Annotations) {
			if err := con.Writer().UpdateDeployItem(ctx, read_write_layer.W000043, di); err != nil {
				logger.Error(err, "unable to update deploy item")
				return reconcile.Result{}, err
			}
		}
	}

	// detect progressing timeout
	// only do something if progressing timeout detection is neither deactivated on the deploy item, nor defaulted by the deploy item and deactivated by default
	if !((di.Spec.Timeout != nil && di.Spec.Timeout.Duration == 0) || (di.Spec.Timeout == nil && con.defaultTimeout == 0)) {
		logger.Debug("check for progressing timeout")
		tmp, err := con.detectProgressingTimeouts(ctx, di)
		if err != nil {
			return reconcile.Result{}, err
		}
		if requeue == nil {
			requeue = tmp
		} else if tmp != nil && *tmp < *requeue {
			requeue = tmp
		}
		if !reflect.DeepEqual(old.Annotations, di.Annotations) {
			if err := con.Writer().UpdateDeployItem(ctx, read_write_layer.W000042, di); err != nil {
				logger.Error(err, "unable to update deploy item")
				return reconcile.Result{}, err
			}
		}
	}

	if requeue == nil {
		return reconcile.Result{}, nil
	}
	logger.Debug("requeue deploy item", "after", requeue.String())
	return reconcile.Result{RequeueAfter: *requeue}, nil
}

func (con *controller) detectPickupTimeouts(ctx context.Context, di *lsv1alpha1.DeployItem) (*time.Duration, error) {
	logger, _ := logging.FromContextOrNew(ctx, nil)
	logger = logger.WithValues(lc.KeyMethod, "detectPickupTimeouts")

	if di.Status.Phase == lsv1alpha1.ExecutionPhaseFailed &&
		di.Status.ObservedGeneration == di.Generation &&
		di.Status.LastError != nil &&
		di.Status.LastError.Reason == lsv1alpha1.PickupTimeoutReason {
		// don't do anything if phase is already failed due to a recent pickup timeout
		// to avoid multiple simultaneous reconciles which would cause further reconciles in the deployers
		logger.Debug("deploy item already failed due to pickup timeout, nothing to do")
		return nil, nil
	}

	if !metav1.HasAnnotation(di.ObjectMeta, string(lsv1alpha1helper.ReconcileTimestamp)) {
		logger.Debug("deploy item doesn't have reconcile timestamp annotation, nothing to do")
		return nil, nil
	}

	ts, err := lsv1alpha1helper.GetTimestampAnnotation(di.ObjectMeta, lsv1alpha1helper.ReconcileTimestamp)
	if err != nil {
		return nil, fmt.Errorf("unable to parse reconcile timestamp annotation: %w", err)
	}
	waitingForPickupDuration := time.Since(ts)
	if waitingForPickupDuration >= con.pickupTimeout {
		// no deployer has picked up the deploy item within the timeframe
		// => pickup timeout
		logger.Debug("pickup timeout occurred")
		di.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
		di.Status.ObservedGeneration = di.Generation
		di.Status.LastError = lserrors.UpdatedError(di.Status.LastError,
			lsv1alpha1.PickupTimeoutOperation,
			lsv1alpha1.PickupTimeoutReason,
			fmt.Sprintf("no deployer has reconciled this deployitem within %d seconds", con.pickupTimeout/time.Second),
			lsv1alpha1.ErrorTimeout,
		)
		return nil, nil
	}

	// deploy item neither picked up nor timed out
	// => requeue shortly after expected timeout
	requeue := con.pickupTimeout - waitingForPickupDuration + (5 * time.Second)
	return &requeue, nil
}

func (con *controller) detectAbortingTimeouts(ctx context.Context, di *lsv1alpha1.DeployItem) (*time.Duration, error) {
	logger, _ := logging.FromContextOrNew(ctx, nil)
	logger = logger.WithValues(lc.KeyMethod, "detectAbortingTimeouts")

	if di.Status.Phase == lsv1alpha1.ExecutionPhaseFailed &&
		di.Status.ObservedGeneration == di.Generation &&
		di.Status.LastError != nil &&
		di.Status.LastError.Reason == lsv1alpha1.AbortingTimeoutReason {
		// don't do anything if phase is already failed due to a recent aborting timeout
		// to avoid multiple simultaneous reconciles which would cause further reconciles in the deployers
		logger.Debug("deploy item already failed due to aborting timeout, nothing to do")
		// should do nothing if the annotations are already removed.
		lsv1alpha1helper.RemoveAbortOperationAndTimestamp(&di.ObjectMeta)
		return nil, nil
	}

	// no aborting timeout if timestamp is missing or deploy item is in a final phase
	if !metav1.HasAnnotation(di.ObjectMeta, string(lsv1alpha1helper.AbortTimestamp)) || lsv1alpha1helper.IsCompletedExecutionPhase(di.Status.Phase) {
		logger.Debug("deploy item doesn't have abort timestamp annotation or is in a final phase, nothing to do")
		return nil, nil
	}

	ts, err := lsv1alpha1helper.GetTimestampAnnotation(di.ObjectMeta, lsv1alpha1helper.AbortTimestamp)
	if err != nil {
		return nil, fmt.Errorf("unable to parse abort timestamp annotation: %w", err)
	}
	waitingForAbortDuration := time.Since(ts)
	if waitingForAbortDuration >= con.abortingTimeout {
		// deploy item has not been aborted within the timeframe
		// => aborting timeout
		logger.Debug("aborting timeout occurred")
		lsv1alpha1helper.RemoveAbortOperationAndTimestamp(&di.ObjectMeta)
		di.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
		di.Status.ObservedGeneration = di.Generation
		di.Status.LastError = lserrors.UpdatedError(di.Status.LastError,
			lsv1alpha1.AbortingTimeoutOperation,
			lsv1alpha1.AbortingTimeoutReason,
			fmt.Sprintf("deployer has not aborted progressing this deploy item within %d seconds",
				con.abortingTimeout/time.Second),
			lsv1alpha1.ErrorTimeout)
		return nil, nil
	}

	// deploy item neither aborted nor timed out
	// => requeue shortly after expected timeout
	requeue := con.abortingTimeout - waitingForAbortDuration + (5 * time.Second)
	return &requeue, nil
}

func (con *controller) detectProgressingTimeouts(ctx context.Context, di *lsv1alpha1.DeployItem) (*time.Duration, error) {
	logger, _ := logging.FromContextOrNew(ctx, nil)
	logger = logger.WithValues(lc.KeyMethod, "detectProgressingTimeouts")

	// no progressing timeout if timestamp is zero or deploy item is in a final phase
	if di.Status.LastReconcileTime.IsZero() || lsv1alpha1helper.IsCompletedExecutionPhase(di.Status.Phase) {
		logger.Debug("deploy item is reconciled for the first time or in a final phase, nothing to do")
		return nil, nil
	}

	var progressingTimeout time.Duration
	if di.Spec.Timeout == nil { // timeout not specified in deploy item, use global default
		progressingTimeout = con.defaultTimeout
	} else {
		progressingTimeout = di.Spec.Timeout.Duration
	}
	progressingDuration := time.Since(di.Status.LastReconcileTime.Time)
	if progressingDuration >= progressingTimeout {
		// the deployer has not finished processing this deploy item within the timeframe
		// => abort it
		logger.Debug("deploy item timed out, setting abort operation annotation")
		lsv1alpha1helper.SetAbortOperationAndTimestamp(&di.ObjectMeta)
		return nil, nil
	}

	// deploy item not yet timed out
	// => requeue shortly after expected timeout
	requeue := progressingTimeout - progressingDuration + (5 * time.Second)
	return &requeue, nil
}

func (con *controller) Writer() *read_write_layer.Writer {
	return read_write_layer.NewWriter(con.c)
}

func HasBeenPickedUp(di *lsv1alpha1.DeployItem) bool {
	return di.Status.LastReconcileTime != nil && !di.Status.LastReconcileTime.Before(di.Status.JobIDGenerationTime)
}
