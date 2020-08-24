// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wait

import (
	"context"
	"math"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/pkg/apis/deployer/container"
	"github.com/gardener/landscaper/pkg/kubernetes"
	kubernetesutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// WaitUntilMainContainerFinished waits until the main container of the pod has finished.
// For a comparison of different possibilities to wait for a container to finish
// see the argo doc: https://github.com/argoproj/argo/blob/master/docs/workflow-executors.md
// This method currently uses the k8s api method for simplicity and stability reasons.
func WaitUntilMainContainerFinished(ctx context.Context, log logr.Logger, podKey client.ObjectKey, defaultBackoff wait.Backoff) error {
	restConfig, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return err
	}

	var kubeClient client.Client
	if err := wait.ExponentialBackoff(defaultBackoff, func() (bool, error) {
		var err error
		kubeClient, err = client.New(restConfig, client.Options{
			Scheme: kubernetes.LandscaperScheme,
		})
		if err != nil {
			log.Error(err, "unable to build kubernetes client")
			return false, nil
		}
		return true, nil
	}); err != nil {
		return err
	}

	backoff := wait.Backoff{
		Duration: 30 * time.Second,
		Factor:   1.25,
		Steps:    math.MaxInt32,
		Cap:      5 * time.Minute,
	}
	// no timeout is needed as we use the max active seconds of the pod to react on the timeout
	return wait.ExponentialBackoff(backoff, func() (bool, error) {
		pod := &corev1.Pod{}
		if err := kubeClient.Get(ctx, podKey, pod); err != nil {
			if apierrors.IsNotFound(err) {
				return false, err
			}
			log.Error(err, "unable to get pod", "pod", podKey.String())
			return false, nil
		}

		mainContainerStatus, err := kubernetesutil.GetStatusForContainer(pod.Status.ContainerStatuses, container.MainContainerName)
		if err != nil {
			log.Error(err, "unable to get container status for main container")
			return false, nil
		}

		if mainContainerStatus.State.Terminated == nil {
			log.V(3).Info("main container is still running...")
			return false, nil
		}
		return true, nil
	})
}
