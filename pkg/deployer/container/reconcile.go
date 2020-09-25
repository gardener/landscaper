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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/apis/deployer/container"
	containerv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	kubernetesutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// Reconcile handles the reconcile flow for a container deploy item.
// todo: do retries on failure: difference between main container failure and init/wait container failure
func (c *Container) Reconcile(ctx context.Context, operation container.OperationType) error {
	pod, err := c.getPod(ctx)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	// do nothing if the pod is still running
	if pod != nil {
		if pod.Status.Phase == corev1.PodPending || pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodUnknown {
			c.DeployItem.Status.Conditions = setConditionsFromPod(pod, c.DeployItem.Status.Conditions)
			return c.kubeClient.Status().Update(ctx, c.DeployItem)
		}
	}

	if c.DeployItem.Status.ObservedGeneration != c.DeployItem.Generation {
		// ensure new pod
		if err := c.ensureServiceAccounts(ctx); err != nil {
			return err
		}
		c.ProviderStatus = &containerv1alpha1.ProviderStatus{}
		podOpts := PodOptions{
			ProviderConfiguration:             c.ProviderConfiguration,
			InitContainer:                     c.Configuration.InitContainer,
			WaitContainer:                     c.Configuration.WaitContainer,
			InitContainerServiceAccountSecret: c.InitContainerServiceAccountSecret,
			WaitContainerServiceAccountSecret: c.WaitContainerServiceAccountSecret,

			Name:      c.DeployItem.Name,
			Namespace: c.DeployItem.Namespace,

			Operation: operation,
			Debug:     true,
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
		c.ProviderStatus.PodStatus.PodName = pod.Name
		c.ProviderStatus.LastOperation = string(operation)
		if err := setStatusFromPod(pod, c.ProviderStatus); err != nil {
			return err
		}
		encStatus, err := EncodeProviderStatus(c.ProviderStatus)
		if err != nil {
			return err
		}

		c.DeployItem.Status.ProviderStatus = encStatus
		c.DeployItem.Status.ObservedGeneration = c.DeployItem.Generation
		c.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseProgressing
		if operation == container.OperationDelete {
			c.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseDeleting
		}
		return c.kubeClient.Status().Update(ctx, c.DeployItem)
	}

	if pod == nil {
		return nil
	}

	if pod.Status.Phase == corev1.PodSucceeded {
		c.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
	}

	if pod.Status.Phase == corev1.PodFailed {
		c.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
	}

	// wait for container to finish
	c.ProviderStatus.LastOperation = string(operation)
	if err := setStatusFromPod(pod, c.ProviderStatus); err != nil {
		return err
	}
	encStatus, err := EncodeProviderStatus(c.ProviderStatus)
	if err != nil {
		return err
	}

	c.DeployItem.Status.ProviderStatus = encStatus
	c.DeployItem.Status.Conditions = setConditionsFromPod(pod, c.DeployItem.Status.Conditions)
	if err := c.kubeClient.Status().Update(ctx, c.DeployItem); err != nil {
		return err
	}

	// only remove the finalizer if we catched the status of the pod
	controllerutil.RemoveFinalizer(pod, container.ContainerDeployerFinalizer)
	if err := c.kubeClient.Update(ctx, pod); err != nil {
		return err
	}
	return nil
}

func setStatusFromPod(pod *corev1.Pod, providerStatus *containerv1alpha1.ProviderStatus) error {
	containerStatus, err := kubernetesutil.GetStatusForContainer(pod.Status.ContainerStatuses, container.MainContainerName)
	if err != nil {
		return nil
	}
	providerStatus.PodStatus.Image = containerStatus.Image
	providerStatus.PodStatus.ImageID = containerStatus.ImageID

	if containerStatus.State.Waiting != nil {
		providerStatus.PodStatus.Message = containerStatus.State.Waiting.Message
		providerStatus.PodStatus.Reason = containerStatus.State.Waiting.Reason
	}
	if containerStatus.State.Running != nil {
		providerStatus.PodStatus.Reason = "Running"
	}
	if containerStatus.State.Terminated != nil {
		providerStatus.PodStatus.Reason = containerStatus.State.Terminated.Reason
		providerStatus.PodStatus.Message = containerStatus.State.Terminated.Message
		providerStatus.PodStatus.ExitCode = &containerStatus.State.Terminated.ExitCode
	}
	return nil
}

func setConditionsFromPod(pod *corev1.Pod, conditions []lsv1alpha1.Condition) []lsv1alpha1.Condition {
	initStatus, err := kubernetesutil.GetStatusForContainer(pod.Status.InitContainerStatuses, container.InitContainerName)
	if err == nil {
		cond := lsv1alpha1helper.GetOrInitCondition(conditions, container.InitContainerConditionType)
		if initStatus.State.Waiting != nil {
			cond = lsv1alpha1helper.UpdatedCondition(cond,
				lsv1alpha1.ConditionProgressing, initStatus.State.Waiting.Reason, initStatus.State.Waiting.Message)
		}
		if initStatus.State.Running != nil {
			cond = lsv1alpha1helper.UpdatedCondition(cond,
				lsv1alpha1.ConditionProgressing,
				"Pod running",
				fmt.Sprintf("Pod started running at %s", initStatus.State.Running.StartedAt.String()))
		}
		if initStatus.State.Terminated != nil {
			if initStatus.State.Terminated.ExitCode == 0 {
				cond = lsv1alpha1helper.UpdatedCondition(cond,
					lsv1alpha1.ConditionTrue,
					"ContainerSucceeded",
					"Container successfully finished")
			} else {
				cond = lsv1alpha1helper.UpdatedCondition(cond,
					lsv1alpha1.ConditionFalse,
					initStatus.State.Terminated.Reason,
					initStatus.State.Terminated.Message)
			}
		}
		conditions = lsv1alpha1helper.MergeConditions(conditions, cond)
	}

	waitStatus, err := kubernetesutil.GetStatusForContainer(pod.Status.ContainerStatuses, container.WaitContainerName)
	if err == nil {
		cond := lsv1alpha1helper.GetOrInitCondition(conditions, container.WaitContainerConditionType)
		if waitStatus.State.Waiting != nil {
			cond = lsv1alpha1helper.UpdatedCondition(cond,
				lsv1alpha1.ConditionProgressing, waitStatus.State.Waiting.Reason, waitStatus.State.Waiting.Message)
		}
		if waitStatus.State.Running != nil {
			cond = lsv1alpha1helper.UpdatedCondition(cond,
				lsv1alpha1.ConditionProgressing,
				"Pod running",
				fmt.Sprintf("Pod started running at %s", waitStatus.State.Running.StartedAt.String()))
		}
		if waitStatus.State.Terminated != nil {
			if waitStatus.State.Terminated.ExitCode == 0 {
				cond = lsv1alpha1helper.UpdatedCondition(cond,
					lsv1alpha1.ConditionTrue,
					"ContainerSucceeded",
					"Container successfully finished")
			} else {
				cond = lsv1alpha1helper.UpdatedCondition(cond,
					lsv1alpha1.ConditionFalse,
					waitStatus.State.Terminated.Reason,
					waitStatus.State.Terminated.Message)
			}
		}
		conditions = lsv1alpha1helper.MergeConditions(conditions, cond)
	}

	return conditions
}
