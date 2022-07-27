// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package lib

import (
	"context"
	"fmt"
	"reflect"
	"time"

	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lserrors "github.com/gardener/landscaper/apis/errors"
)

// HandleErrorFunc returns a error handler func for deployers.
// The functions automatically sets the phase for long running errors and updates the status accordingly.
func HandleErrorFunc(ctx context.Context, err lserrors.LsError, log logging.Logger, c client.Client,
	eventRecorder record.EventRecorder, oldDeployItem, deployItem *lsv1alpha1.DeployItem, isDelete bool) error {
	// if successfully deleted we could not update the object
	if isDelete && err == nil {
		di := &lsv1alpha1.DeployItem{}
		if err2 := read_write_layer.GetDeployItem(ctx, c, kutil.ObjectKey(deployItem.Name, deployItem.Namespace), di); err2 != nil {
			if apierrors.IsNotFound(err2) {
				return nil
			}
		}
	}

	deployItem.Status.LastError = lserrors.TryUpdateLsError(deployItem.Status.LastError, err)

	phaseForLastError := lserrors.GetPhaseForLastError(lsv1alpha1.ComponentInstallationPhase(deployItem.Status.Phase),
		deployItem.Status.LastError, 5*time.Minute)
	deployItem.Status.Phase = lsv1alpha1.ExecutionPhase(phaseForLastError)

	if deployItem.Status.LastError != nil {
		lastErr := deployItem.Status.LastError
		eventRecorder.Event(deployItem, corev1.EventTypeWarning, lastErr.Reason, lastErr.Message)
	}

	if !reflect.DeepEqual(oldDeployItem.Status, deployItem.Status) {
		writer := read_write_layer.NewWriter(log, c)
		if err2 := writer.UpdateDeployItemStatus(ctx, read_write_layer.W000051, deployItem); err2 != nil {
			if apierrors.IsConflict(err2) { // reduce logging
				log.Logr().V(5).Info(fmt.Sprintf("unable to update status: %s", err2.Error()))
			} else {
				log.Error(err2, "unable to update status")
			}
			// retry on conflict
			if err == nil {
				return err2
			}
		}
	}
	return err
}
