// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployitem

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lscore "github.com/gardener/landscaper/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
)

const (
	PickupTimeoutReason      = "PickupTimeout"    // for error messages
	PickupTimeoutOperation   = "WaitingForPickup" // for error messages
	AbortingTimeoutReason    = "AbortingTimeout"  // for error messages
	AbortingTimeoutOperation = "WaitingForAbort"  // for error messages
)

// NewController creates a new deploy item controller that handles timeouts
// To detect pickup timeouts (when a DeployItem resource is not reconciled by any deployer within a specified timeframe), the controller checks for a timestamp annotation.
// It is expected that deployers remove the timestamp annotation from deploy items during reconciliation. If the timestamp annotation exists and is older than a specified duration,
// the controller marks the deploy item as failed.
// pickupTimeout is a string containing the pickup timeout duration, either as 'none' or as a duration that can be parsed by time.ParseDuration.
func NewController(log logr.Logger, c client.Client, scheme *runtime.Scheme, pickupTimeout, abortingTimeout, defaultTimeout lscore.Duration) (reconcile.Reconciler, error) {
	con := controller{log: log, c: c, scheme: scheme}
	con.pickupTimeout = pickupTimeout.Duration
	con.abortingTimeout = abortingTimeout.Duration
	con.defaultTimeout = defaultTimeout.Duration

	// log pickup timeout
	log.Info("deploy item pickup timeout detection", "active", con.pickupTimeout != 0, "timeout", con.pickupTimeout.String())
	log.Info("deploy item aborting timeout detection", "active", con.abortingTimeout != 0, "timeout", con.abortingTimeout.String())
	log.Info("deploy item default timeout", "active", con.defaultTimeout != 0, "timeout", con.defaultTimeout.String())

	return &con, nil
}

type controller struct {
	log             logr.Logger
	c               client.Client
	scheme          *runtime.Scheme
	pickupTimeout   time.Duration
	abortingTimeout time.Duration
	defaultTimeout  time.Duration
}

func (con *controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := con.log.WithValues("resource", req.NamespacedName.String())
	logger.Info("reconcile")

	di := &lsv1alpha1.DeployItem{}
	if err := con.c.Get(ctx, req.NamespacedName, di); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	var requeue *time.Duration
	var err error
	old := di.DeepCopy()

	// detect pickup timeout
	if con.pickupTimeout != 0 {
		logger.V(5).Info("check for pickup timeout")
		requeue, err = con.detectPickupTimeouts(logger, di)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	if !reflect.DeepEqual(old.Status, di.Status) {
		if err := con.c.Status().Update(ctx, di); err != nil {
			logger.Error(err, "unable to set deployitem status")
			return reconcile.Result{}, err
		}
		// if there was a pickup timeout, no need to check for anything else
		return reconcile.Result{}, nil
	}

	// detect aborting timeout
	if con.abortingTimeout != 0 {
		logger.V(5).Info("check for aborting timeout")
		tmp, err := con.detectAbortingTimeouts(logger, di)
		if err != nil {
			return reconcile.Result{}, err
		}
		if requeue == nil {
			requeue = tmp
		} else if tmp != nil && *tmp < *requeue {
			requeue = tmp
		}
	}
	if !reflect.DeepEqual(old.Status, di.Status) {
		if err := con.c.Status().Update(ctx, di); err != nil {
			logger.Error(err, "unable to set deployitem status")
			return reconcile.Result{}, err
		}
		// if there was an aborting timeout, no need to check for anything else
		return reconcile.Result{}, nil
	}

	// detect progressing timeout
	// only do something if progressing timeout detection is neither deactivated on the deploy item, nor defaulted by the deploy item and deactivated by default
	if !((di.Spec.Timeout != nil && di.Spec.Timeout.Duration == 0) || (di.Spec.Timeout == nil && con.defaultTimeout == 0)) {
		logger.V(5).Info("check for progressing timeout")
		tmp, err := con.detectProgressingTimeouts(logger, di)
		if err != nil {
			return reconcile.Result{}, err
		}
		if requeue == nil {
			requeue = tmp
		} else if tmp != nil && *tmp < *requeue {
			requeue = tmp
		}
	}
	if !reflect.DeepEqual(old.Annotations, di.Annotations) {
		if err := con.c.Update(ctx, di); err != nil {
			logger.Error(err, "unable to update deploy item")
			return reconcile.Result{}, err
		}
	}

	if requeue == nil {
		return reconcile.Result{}, nil
	}
	logger.V(5).Info("requeue deploy item", "after", requeue.String())
	return reconcile.Result{RequeueAfter: *requeue}, nil
}

func (con *controller) detectPickupTimeouts(log logr.Logger, di *lsv1alpha1.DeployItem) (*time.Duration, error) {
	logger := log.WithValues("operation", "DetectPickupTimeouts")
	if di.Status.Phase == lsv1alpha1.ExecutionPhaseFailed && di.Status.LastError != nil && di.Status.LastError.Reason == PickupTimeoutReason {
		// don't do anything if phase is already failed due to a recent pickup timeout
		// to avoid multiple simultaneous reconciles which would cause further reconciles in the deployers
		logger.V(7).Info("deploy item already failed due to pickup timeout, nothing to do")
		return nil, nil
	}

	if !lsv1alpha1helper.HasTimestampAnnotation(di.ObjectMeta, lsv1alpha1helper.ReconcileTimestamp) {
		logger.V(7).Info("deploy item doesn't have reconcile timestamp annotation, nothing to do")
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
		logger.V(5).Info("pickup timeout occurred")
		di.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
		di.Status.LastError = lsv1alpha1helper.UpdatedError(di.Status.LastError, PickupTimeoutOperation, PickupTimeoutReason, fmt.Sprintf("no deployer has reconciled this deployitem within %d seconds", con.pickupTimeout/time.Second), lsv1alpha1.ErrorTimeout)
		return nil, nil
	}

	// deploy item neither picked up nor timed out
	// => requeue shortly after expected timeout
	requeue := con.pickupTimeout - waitingForPickupDuration + (5 * time.Second)
	return &requeue, nil
}

func (con *controller) detectAbortingTimeouts(log logr.Logger, di *lsv1alpha1.DeployItem) (*time.Duration, error) {
	logger := log.WithValues("operation", "DetectAbortingTimeouts")
	if di.Status.Phase == lsv1alpha1.ExecutionPhaseFailed && di.Status.LastError != nil && di.Status.LastError.Reason == AbortingTimeoutReason {
		// don't do anything if phase is already failed due to a recent aborting timeout
		// to avoid multiple simultaneous reconciles which would cause further reconciles in the deployers
		logger.V(7).Info("deploy item already failed due to aborting timeout, nothing to do")
		return nil, nil
	}

	// no aborting timeout if timestamp is missing or deploy item is in a final phase
	if !lsv1alpha1helper.HasTimestampAnnotation(di.ObjectMeta, lsv1alpha1helper.AbortTimestamp) || di.Status.Phase == lsv1alpha1.ExecutionPhaseSucceeded || di.Status.Phase == lsv1alpha1.ExecutionPhaseFailed {
		logger.V(7).Info("deploy item doesn't have abort timestamp annotation or is in a final phase, nothing to do")
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
		logger.V(5).Info("aborting timeout occurred")
		di.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
		di.Status.LastError = lsv1alpha1helper.UpdatedError(di.Status.LastError, AbortingTimeoutOperation, AbortingTimeoutReason, fmt.Sprintf("deployer has not aborted progressing this deploy item within %d seconds", con.abortingTimeout/time.Second), lsv1alpha1.ErrorTimeout)
		return nil, nil
	}

	// deploy item neither aborted nor timed out
	// => requeue shortly after expected timeout
	requeue := con.abortingTimeout - waitingForAbortDuration + (5 * time.Second)
	return &requeue, nil
}

func (con *controller) detectProgressingTimeouts(log logr.Logger, di *lsv1alpha1.DeployItem) (*time.Duration, error) {
	logger := log.WithValues("operation", "DetectProgressingTimeouts")
	// no progressing timeout if timestamp is zero or deploy item is in a final phase
	if di.Status.LastChangeReconcileTime.IsZero() || di.Status.Phase == lsv1alpha1.ExecutionPhaseSucceeded || di.Status.Phase == lsv1alpha1.ExecutionPhaseFailed {
		logger.V(7).Info("deploy item is reconciled for the first time or in a final phase, nothing to do")
		return nil, nil
	}

	var progressingTimeout time.Duration
	if di.Spec.Timeout == nil { // timeout not specified in deploy item, use global default
		progressingTimeout = con.defaultTimeout
	} else {
		progressingTimeout = di.Spec.Timeout.Duration
	}
	progressingDuration := time.Since(di.Status.LastChangeReconcileTime.Time)
	if progressingDuration >= progressingTimeout {
		// the deployer has not finished processing this deploy item within the timeframe
		// => abort it
		logger.V(5).Info("deploy item timed out, setting abort operation annotation")
		lsv1alpha1helper.SetAbortOperationAndTimestamp(&di.ObjectMeta)
		return nil, nil
	}

	// deploy item not yet timed out
	// => requeue shortly after expected timeout
	requeue := progressingTimeout - progressingDuration + (5 * time.Second)
	return &requeue, nil
}
