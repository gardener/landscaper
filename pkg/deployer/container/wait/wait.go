// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package wait

import (
	"context"
	"math"
	"time"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/apis/deployer/container"
	kubernetesutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
)

// WaitUntilMainContainerFinished waits until the main container of the pod has finished.
// For a comparison of different possibilities to wait for a container to finish
// see the argo doc: https://github.com/argoproj/argo/blob/master/docs/workflow-executors.md
// This method currently uses the k8s api method for simplicity and stability reasons.
func WaitUntilMainContainerFinished(ctx context.Context, kubeClient client.Client, podKey client.ObjectKey) error {
	log, ctx := logging.FromContextOrNew(ctx, nil)
	backoff := wait.Backoff{
		Duration: 30 * time.Second,
		Factor:   1.25,
		Steps:    math.MaxInt32,
		Cap:      5 * time.Minute,
	}
	// no timeout is needed as we use the max active seconds of the pod to react on the timeout
	return wait.ExponentialBackoff(backoff, func() (bool, error) {
		pod := &corev1.Pod{}
		if err := read_write_layer.GetPod(ctx, kubeClient, podKey, pod, read_write_layer.R000041); err != nil {
			if apierrors.IsNotFound(err) {
				return false, err
			}
			if apierrors.IsUnauthorized(err) {
				return false, err
			}
			log.Error(err, "Unable to get pod", lc.KeyResource, podKey.String())
			return false, nil
		}

		mainContainerStatus, err := kubernetesutil.GetStatusForContainer(pod.Status.ContainerStatuses, container.MainContainerName)
		if err != nil {
			log.Error(err, "Unable to get container status for main container")
			return false, nil
		}

		if mainContainerStatus.State.Terminated == nil {
			log.Debug("Main container is still running...")
			return false, nil
		}
		return true, nil
	})
}
