// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package lib

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
)

// HandleErrorFunc returns a error handler func for deployers.
// The functions automatically sets the phase for long running errors and updates the status accordingly.
func HandleErrorFunc(log logr.Logger, client client.Client, deployItem *lsv1alpha1.DeployItem) func(ctx context.Context, err error) error {
	old := deployItem.DeepCopy()
	return func(ctx context.Context, err error) error {
		deployItem.Status.LastError = lsv1alpha1helper.TryUpdateError(deployItem.Status.LastError, err)
		deployItem.Status.Phase = lsv1alpha1.ExecutionPhase(lsv1alpha1helper.GetPhaseForLastError(
			lsv1alpha1.ComponentInstallationPhase(deployItem.Status.Phase),
			deployItem.Status.LastError,
			5*time.Minute))
		if !reflect.DeepEqual(old.Status, deployItem.Status) {
			if err2 := client.Status().Update(ctx, deployItem); err2 != nil {
				if apierrors.IsConflict(err2) { // reduce logging
					log.V(5).Info(fmt.Sprintf("unable to update status: %s", err2.Error()))
				} else {
					log.Error(err2, "unable to update status")
				}
				// retry on conflict
				if err != nil {
					return err2
				}
			}
		}
		return err
	}
}
