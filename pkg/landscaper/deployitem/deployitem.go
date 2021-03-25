// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployitem

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
)

const (
	DefaultPickupTimeout   = 5 * time.Minute    // default pickup timeout
	PickupTimeoutReason    = "PickupTimeout"    // for error messages
	PickupTimeoutOperation = "WaitingForPickup" // for error messages
)

// FailDueToTimeout sets the phase of the deployitem to failed and sets lastError to a timeout error
func FailDueToTimeout(ctx context.Context, c client.Client, log logr.Logger, di *lsv1alpha1.DeployItem, timeoutMessage string) error {
	di.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
	di.Status.LastError = lsv1alpha1helper.UpdatedError(di.Status.LastError, PickupTimeoutOperation, PickupTimeoutReason, timeoutMessage, lsv1alpha1.ErrorTimeout)
	if err := c.Status().Update(ctx, di); err != nil {
		log.Error(err, "unable to set deployitem status")
		return err
	}
	return nil
}
