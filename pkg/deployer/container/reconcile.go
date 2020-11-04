// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

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
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// Reconcile handles the reconcile flow for a container deploy item.
// todo: do retries on failure: difference between main container failure and init/wait container failure
func (c *Container) Reconcile(ctx context.Context, operation container.OperationType) error {
	pod, err := c.getPod(ctx)
	if err != nil && !apierrors.IsNotFound(err) {
		c.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(c.DeployItem.Status.LastError,
			"Reconcile", "FetchRunningPod", err.Error())
		return err
	}

	// do nothing if the pod is still running
	if pod != nil {
		c.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseProgressing
		if pod.Status.Phase == corev1.PodPending || pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodUnknown {
			c.DeployItem.Status.Conditions = setConditionsFromPod(pod, c.DeployItem.Status.Conditions)
			return c.lsClient.Status().Update(ctx, c.DeployItem)
		}
	}

	if c.DeployItem.Status.ObservedGeneration != c.DeployItem.Generation || lsv1alpha1helper.HasOperation(c.DeployItem.ObjectMeta, lsv1alpha1.ReconcileOperation) {
		operationName := "DeployPod"
		if err := c.SyncConfiguration(ctx); err != nil {
			c.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(c.DeployItem.Status.LastError,
				operationName, "SyncConfiguration", err.Error())
			return err
		}
		// ensure new pod
		if err := c.ensureServiceAccounts(ctx); err != nil {
			c.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(c.DeployItem.Status.LastError,
				operationName, "EnsurePodRBAC", err.Error())
			return err
		}
		c.ProviderStatus = &containerv1alpha1.ProviderStatus{}
		podOpts := PodOptions{
			ProviderConfiguration:             c.ProviderConfiguration,
			InitContainer:                     c.Configuration.InitContainer,
			WaitContainer:                     c.Configuration.WaitContainer,
			InitContainerServiceAccountSecret: c.InitContainerServiceAccountSecret,
			WaitContainerServiceAccountSecret: c.WaitContainerServiceAccountSecret,
			ConfigurationSecretName:           ConfigurationSecretName(c.DeployItem.Namespace, c.DeployItem.Name),

			Name:                c.DeployItem.Name,
			Namespace:           c.Configuration.Namespace,
			DeployItemName:      c.DeployItem.Name,
			DeployItemNamespace: c.DeployItem.Namespace,

			Operation: operation,
			Debug:     true,
		}
		pod, err := generatePod(podOpts)
		if err != nil {
			c.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(c.DeployItem.Status.LastError,
				operationName, "PodGeneration", err.Error())
			return err
		}

		if err := c.hostClient.Create(ctx, pod); err != nil {
			c.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(c.DeployItem.Status.LastError,
				operationName, "CreatePod", err.Error())
			return err
		}

		// update status
		c.ProviderStatus.PodStatus.PodName = pod.Name
		c.ProviderStatus.LastOperation = string(operation)
		if err := setStatusFromPod(pod, c.ProviderStatus); err != nil {
			c.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(c.DeployItem.Status.LastError,
				operationName, "UpdatePodStatus", err.Error())
			return err
		}
		encStatus, err := EncodeProviderStatus(c.ProviderStatus)
		if err != nil {
			c.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(c.DeployItem.Status.LastError,
				operationName, "EncodeProviderStatus", err.Error())
			return err
		}

		c.DeployItem.Status.ProviderStatus = encStatus
		c.DeployItem.Status.ObservedGeneration = c.DeployItem.Generation
		c.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseProgressing
		if operation == container.OperationDelete {
			c.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseDeleting
		}
		if err := c.lsClient.Status().Update(ctx, c.DeployItem); err != nil {
			c.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(c.DeployItem.Status.LastError,
				operationName, "UpdateDeployItemStatus", err.Error())
			return err
		}

		if lsv1alpha1helper.HasOperation(c.DeployItem.ObjectMeta, lsv1alpha1.ReconcileOperation) {
			delete(c.DeployItem.Annotations, lsv1alpha1.OperationAnnotation)
			if err := c.lsClient.Update(ctx, c.DeployItem); err != nil {
				c.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(c.DeployItem.Status.LastError,
					operationName, "RemoveReconcileAnnotation", err.Error())
				return err
			}
		}
		return nil
	}

	if pod == nil {
		return nil
	}
	operationName := "Complete"

	if pod.Status.Phase == corev1.PodSucceeded {
		if err := c.SyncExport(ctx); err != nil {
			c.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(c.DeployItem.Status.LastError,
				operationName, "SyncExport", err.Error())
			return err
		}
		c.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
	}
	if pod.Status.Phase == corev1.PodFailed {
		c.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
	}

	c.ProviderStatus.LastOperation = string(operation)
	if err := setStatusFromPod(pod, c.ProviderStatus); err != nil {
		c.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(c.DeployItem.Status.LastError,
			operationName, "FetchPodStatus", err.Error())
		return err
	}

	encStatus, err := EncodeProviderStatus(c.ProviderStatus)
	if err != nil {
		c.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(c.DeployItem.Status.LastError,
			operationName, "SetProviderStatus", err.Error())
		return err
	}

	c.DeployItem.Status.ProviderStatus = encStatus
	c.DeployItem.Status.Conditions = setConditionsFromPod(pod, c.DeployItem.Status.Conditions)
	if err := c.lsClient.Status().Update(ctx, c.DeployItem); err != nil {
		c.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(c.DeployItem.Status.LastError,
			operationName, "UpdateDeployItemStatus", err.Error())
		return err
	}

	// only remove the finalizer if we get the status of the pod
	controllerutil.RemoveFinalizer(pod, container.ContainerDeployerFinalizer)
	if err := c.hostClient.Update(ctx, pod); err != nil {
		c.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(c.DeployItem.Status.LastError,
			operationName, "RemoveFinalizer", err.Error())
		return err
	}
	c.DeployItem.Status.LastError = nil
	return nil
}

func setStatusFromPod(pod *corev1.Pod, providerStatus *containerv1alpha1.ProviderStatus) error {
	containerStatus, err := kutil.GetStatusForContainer(pod.Status.ContainerStatuses, container.MainContainerName)
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
	initStatus, err := kutil.GetStatusForContainer(pod.Status.InitContainerStatuses, container.InitContainerName)
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

	waitStatus, err := kutil.GetStatusForContainer(pod.Status.ContainerStatuses, container.WaitContainerName)
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

// SyncConfiguration syncs the provider configuration data as secret to the host cluster.
func (c *Container) SyncConfiguration(ctx context.Context) error {
	secret := &corev1.Secret{}
	secret.Name = ConfigurationSecretName(c.DeployItem.Namespace, c.DeployItem.Name)
	secret.Namespace = c.Configuration.Namespace
	if _, err := controllerutil.CreateOrUpdate(ctx, c.hostClient, secret, func() error {
		kutil.SetMetaDataLabel(&secret.ObjectMeta, container.ContainerDeployerNameLabel, c.DeployItem.Name)
		secret.Data = map[string][]byte{
			container.ConfigurationFilename: c.DeployItem.Spec.Configuration.Raw,
		}
		return nil
	}); err != nil {
		return fmt.Errorf("unable to sync provider configuration to host cluster: %w", err)
	}
	return nil
}

// SyncExport syncs the export secret from the wait container to the deploy item export.
func (c *Container) SyncExport(ctx context.Context) error {
	c.log.V(3).Info("Sync export to landscaper cluster", "deployitem", kutil.ObjectKey(c.DeployItem.Name, c.DeployItem.Namespace).String())
	secret := &corev1.Secret{}
	key := kutil.ObjectKey(DeployItemExportSecretName(c.DeployItem.Name), c.Configuration.Namespace)
	if err := c.hostClient.Get(ctx, key, secret); err != nil {
		if apierrors.IsNotFound(err) {
			c.log.Info("no export found for deploy item", "deployitem", key.String())
			return nil
		}
		return fmt.Errorf("unable to fetch exported secret %s from host cluster: %w", ExportSecretName(c.DeployItem.Namespace, c.DeployItem.Name), err)
	}

	expSecret := &corev1.Secret{}
	expSecret.Name = secret.Name
	expSecret.Namespace = secret.Namespace
	if _, err := controllerutil.CreateOrUpdate(ctx, c.lsClient, expSecret, func() error {
		expSecret.Data = secret.Data
		return controllerutil.SetControllerReference(c.DeployItem, expSecret, kubernetes.LandscaperScheme)
	}); err != nil {
		return fmt.Errorf("unable to sync export to landscaper cluster: %w", err)
	}

	c.DeployItem.Status.ExportReference = &lsv1alpha1.ObjectReference{
		Name:      expSecret.Name,
		Namespace: expSecret.Namespace,
	}

	return nil
}
