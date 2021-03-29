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
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
)

const (
	PickupTimeoutReason    = "PickupTimeout"    // for error messages
	PickupTimeoutOperation = "WaitingForPickup" // for error messages
)

// NewController creates a new deploy item controller that handles timeouts
// To detect pickup timeouts (when a DeployItem resource is not reconciled by any deployer within a specified timeframe), the controller checks for a timestamp annotation.
// It is expected that deployers remove the timestamp annotation from deploy items during reconciliation. If the timestamp annotation exists and is older than a specified duration,
// the controller marks the deploy item as failed.
// rawPickupTimeout is a string containing the pickup timeout duration, either as 'none' or as a duration that can be parsed by time.ParseDuration.
func NewController(log logr.Logger, c client.Client, scheme *runtime.Scheme, rawPickupTimeout string) (reconcile.Reconciler, error) {
	con := controller{log: log, c: c, scheme: scheme}
	if rawPickupTimeout == "none" {
		con.pickupTimeout = nil
	} else {
		tmp, err := time.ParseDuration(rawPickupTimeout)
		if err != nil {
			return nil, fmt.Errorf("unable to parse deploy item pickup timeout into a duration: %w", err)
		}
		con.pickupTimeout = &tmp
	}

	// log pickup timeout
	timeoutLog := ""
	if con.pickupTimeout != nil {
		timeoutLog = con.pickupTimeout.String()
	}
	log.Info("deploy item pickup timeout detection", "active", con.pickupTimeout != nil, "timeout", timeoutLog)

	return &con, nil
}

type controller struct {
	log           logr.Logger
	c             client.Client
	scheme        *runtime.Scheme
	pickupTimeout *time.Duration
}

func (con *controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := con.log.WithValues("resource", req.NamespacedName.String())
	if con.pickupTimeout == nil {
		logger.V(7).Info("skipping reconcile as pickup timeout detection is disabled")
		return reconcile.Result{}, nil
	}
	logger.Info("reconcile")

	di := &lsv1alpha1.DeployItem{}
	if err := con.c.Get(ctx, req.NamespacedName, di); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if di.Status.Phase == lsv1alpha1.ExecutionPhaseFailed && di.Status.LastError != nil && di.Status.LastError.Reason == PickupTimeoutReason {
		// don't do anything if phase is already failed due to a recent pickup timeout
		// to avoid multiple simultaneous reconciles which would cause further reconciles in the deployers
		return reconcile.Result{}, nil
	}

	if !lsv1alpha1helper.HasTimestampAnnotation(di.ObjectMeta, lsv1alpha1helper.ReconcileTimestamp) {
		return reconcile.Result{}, nil
	}

	ts, err := lsv1alpha1helper.GetTimestampAnnotation(di.ObjectMeta, lsv1alpha1helper.ReconcileTimestamp)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("unable to parse timestamp annotation: %w", err)
	}
	waitingForPickupDuration := time.Since(ts)
	if waitingForPickupDuration >= *con.pickupTimeout {
		// no deployer has picked up the deployitem within the timeframe
		// => pickup timeout
		di.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
		di.Status.LastError = lsv1alpha1helper.UpdatedError(di.Status.LastError, PickupTimeoutOperation, PickupTimeoutReason, fmt.Sprintf("no deployer has reconciled this deployitem within %d seconds", *con.pickupTimeout/time.Second), lsv1alpha1.ErrorTimeout)
		if err := con.c.Status().Update(ctx, di); err != nil {
			logger.Error(err, "unable to set deployitem status")
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// deploy item neither picked up nor timed out
	// => requeue shortly after expected timeout
	return reconcile.Result{RequeueAfter: *con.pickupTimeout - waitingForPickupDuration + (5 * time.Second)}, nil

}
