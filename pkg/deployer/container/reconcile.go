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

package container

import (
	"context"
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/apis/deployer/container"
	containerv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	kubernetesutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// Reconcile handles the reconcile flow for a container deploy item.
func (c *Container) Reconcile(ctx context.Context) error {
	if len(c.ProviderStatus.PodName) == 0 {
		c.ProviderStatus = &containerv1alpha1.ProviderStatus{}
		podOpts := PodOptions{
			ProviderConfiguration: c.ProviderConfiguration,
			InitContainer:         c.Configuration.InitContainer,
			SidecarContainer:      c.Configuration.SidecarContainer,

			Name:      c.DeployItem.Name,
			Namespace: c.DeployItem.Namespace,

			Operation:     container.OperationReconcile,
			DefinitionRef: c.DeployItem.Spec.DefinitionRef,
			ImportsRef:    c.DeployItem.Spec.ImportReference,
			OCIConfig:     []byte{},
		}
		pod, err := generatePod(podOpts)
		if err != nil {
			return err
		}

		if err := controllerutil.SetControllerReference(c.DeployItem, pod, kubernetes.LandscaperScheme); err != nil {
			return err
		}

		if err := c.kubeClient.Create(ctx, pod); err != nil {
			return err
		}

		// update status
		c.ProviderStatus.PodName = pod.Name
		containerStatus, _ := kubernetesutil.GetStatusForContainer(pod.Status.ContainerStatuses, "main")
		c.ProviderStatus.Image = containerStatus.Image
		c.ProviderStatus.ImageID = containerStatus.ImageID
		encStatus, err := encodeStatus(c.ProviderStatus)
		if err != nil {
			return err
		}

		c.DeployItem.Status.ProviderStatus = encStatus
		c.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseProgressing
		c.DeployItem.Status.ObservedGeneration = c.DeployItem.Generation
		return c.kubeClient.Status().Update(ctx, c.DeployItem)
	}

	// wait for container to finish
	pod := &corev1.Pod{}
	if err := c.kubeClient.Get(ctx, client.ObjectKey{Name: c.ProviderStatus.PodName, Namespace: c.DeployItem.Namespace}, pod); err != nil {
		if apierrors.IsNotFound(err) {
			// we missed the container deletion, so we have to set the podName to false and retry the operation
			c.log.Info("missed deletion of pod. Retry operation", "deployItem", c.DeployItem.Name, "pod", c.ProviderStatus.PodName)
			c.ProviderStatus.PodName = ""
			encStatus, err := encodeStatus(c.ProviderStatus)
			if err != nil {
				return err
			}

			// todo: set retry condition
			c.DeployItem.Status.ProviderStatus = encStatus
			c.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseInit
			c.DeployItem.Status.ObservedGeneration = c.DeployItem.Generation - 1
			return c.kubeClient.Status().Update(ctx, c.DeployItem)
		}
		return err
	}

	// do nothing if the pod is still running
	if pod.Status.Phase == corev1.PodPending || pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodUnknown {
		return nil
	}

	if pod.Status.Phase == corev1.PodSucceeded {
		controllerutil.RemoveFinalizer(pod, container.ContainerDeployerFinalizer)
		if err := c.kubeClient.Update(ctx, pod); err != nil {
			return err
		}
		c.ProviderStatus.PodName = ""
		encStatus, err := encodeStatus(c.ProviderStatus)
		if err != nil {
			return err
		}
		c.DeployItem.Status.ProviderStatus = encStatus
		c.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
		return c.kubeClient.Status().Update(ctx, c.DeployItem)
	}

	if pod.Status.Phase == corev1.PodFailed {
		c.ProviderStatus.PodName = ""
		c.ProviderStatus.Message = pod.Status.Message
		c.ProviderStatus.Reason = pod.Status.Reason
		encStatus, err := encodeStatus(c.ProviderStatus)
		if err != nil {
			return err
		}
		controllerutil.RemoveFinalizer(pod, container.ContainerDeployerFinalizer)
		if err := c.kubeClient.Update(ctx, pod); err != nil {
			return err
		}

		c.DeployItem.Status.ProviderStatus = encStatus
		c.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
		return c.kubeClient.Status().Update(ctx, c.DeployItem)
	}

	return nil
}

func encodeStatus(status *containerv1alpha1.ProviderStatus) ([]byte, error) {
	status.TypeMeta = metav1.TypeMeta{
		APIVersion: containerv1alpha1.SchemeGroupVersion.String(),
		Kind:       "ProviderStatus",
	}

	return json.Marshal(status)
}
