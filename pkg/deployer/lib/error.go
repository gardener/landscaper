// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package lib

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lserrors "github.com/gardener/landscaper/apis/errors"
)

// HandleErrorFunc returns a error handler func for deployers.
// The functions automatically sets the phase for long running errors and updates the status accordingly.
func HandleErrorFunc(log logr.Logger, client client.Client, eventRecorder record.EventRecorder, deployItem *lsv1alpha1.DeployItem) func(ctx context.Context, err error) error {
	old := deployItem.DeepCopy()
	return func(ctx context.Context, err error) error {
		deployItem.Status.LastError = lserrors.TryUpdateError(deployItem.Status.LastError, err)
		deployItem.Status.Phase = lsv1alpha1.ExecutionPhase(lserrors.GetPhaseForLastError(
			lsv1alpha1.ComponentInstallationPhase(deployItem.Status.Phase),
			deployItem.Status.LastError,
			5*time.Minute))
		if deployItem.Status.LastError != nil {
			lastErr := deployItem.Status.LastError
			eventRecorder.Event(deployItem, corev1.EventTypeWarning, lastErr.Reason, lastErr.Message)
		}

		if !reflect.DeepEqual(old.Status, deployItem.Status) {
			if err2 := read_write_layer.UpdateDeployItemStatus(ctx, read_write_layer.W000051, client, deployItem); err2 != nil {
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
