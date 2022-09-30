// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployitem

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lscore "github.com/gardener/landscaper/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
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

func (con *controller) Writer() *read_write_layer.Writer {
	return read_write_layer.NewWriter(con.c)
}

func HasBeenPickedUp(di *lsv1alpha1.DeployItem) bool {
	return di.Status.LastReconcileTime != nil && !di.Status.LastReconcileTime.Before(di.Status.JobIDGenerationTime)
}
