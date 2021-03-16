// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apimacherrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	terraformv1alpha1 "github.com/gardener/landscaper/apis/deployer/terraform/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/terraform/terraformer"
)

// OperationType defines the value of an Operation that defines the terraform action.
type OperationType string

const (
	// Type defines the type of the execution.
	Type lsv1alpha1.DeployItemType = "landscaper.gardener.cloud/terraform"

	// OperationReconcile is the value of the Operation that defines a reconcile operation.
	OperationReconcile OperationType = "RECONCILE"

	// OperationDelete is the value of the Operation that defines a delete operation.
	OperationDelete OperationType = "DELETE"

	// DeadLinePodNotReady is the time to wait before deleting a not ready pod.
	DeadLinePodNotReady = 10 * time.Minute
)

func (t *Terraform) Reconcile(ctx context.Context, op OperationType) error {
	var (
		currOp  string                    = string(op)
		command string                    = terraformer.ApplyCommand
		phase   lsv1alpha1.ExecutionPhase = lsv1alpha1.ExecutionPhaseProgressing
	)

	// Only destroy when there is no pod applying configuration
	// and the last phase was a success to ensure destroying everything.
	itemPhase := t.DeployItem.Status.Phase
	if (itemPhase == lsv1alpha1.ExecutionPhaseSucceeded) && (op == OperationDelete) || itemPhase == lsv1alpha1.ExecutionPhaseDeleting {
		command = terraformer.DestroyCommand
		phase = lsv1alpha1.ExecutionPhaseDeleting
	}

	tfr := terraformer.New(
		t.log, t.hostClient, t.hostRestConfig,
		t.Configuration.Namespace, t.Configuration.Terraformer.LogLevel,
		t.DeployItem.Namespace, t.DeployItem.Name,
		t.Configuration.Terraformer.InitContainer,
		t.Configuration.Terraformer.TerraformContainer,
	)

	// Check if the Terraformer pod is running.
	pod, err := tfr.GetPod(ctx)
	if client.IgnoreNotFound(err) != nil {
		return lsv1alpha1helper.NewWrappedError(err,
			currOp, "GetTerraformerPod", fmt.Sprintf("unable to get Terraformer pod: %s", err.Error()))
	}

	// Check if the Terraformer pod was deleted between two reconciles.
	if apierrors.IsNotFound(err) && (itemPhase == lsv1alpha1.ExecutionPhaseProgressing || itemPhase == lsv1alpha1.ExecutionPhaseDeleting) {
		t.log.Error(err, "Terraformer pod disappeared unexpectedly, it may have been manually deleted")
	}

	// Nothing is running, a Terraformer pod can be started.
	if pod == nil {
		if err := t.DeployReconcilePod(ctx, currOp, command, tfr, phase); err != nil {
			return lsv1alpha1helper.NewWrappedError(err,
				currOp, "DeployTerraformerPod", fmt.Sprintf("failed to deploy Terraformer pod: %s", err.Error()))
		}
		return nil
	}

	var (
		podCreatedSince                            = time.Since(pod.ObjectMeta.CreationTimestamp.Time)
		podPhase                                   = pod.Status.Phase
		podConditions     []corev1.PodCondition    = pod.Status.Conditions
		containerStatuses []corev1.ContainerStatus = pod.Status.ContainerStatuses
	)

	if !isPodReady(podConditions) && podCreatedSince > (DeadLinePodNotReady) {
		var allErrs []error
		err := fmt.Errorf("terraformer pod has been not ready for more than %s", DeadLinePodNotReady.String())
		allErrs = append(allErrs, err)
		t.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
		t.DeployItem.Status.LastError = lsv1alpha1helper.UpdatedError(t.DeployItem.Status.LastError,
			currOp, "PodNotReadyForTooLong", err.Error())

		// Force deletion of the pod.
		opts := &client.DeleteOptions{GracePeriodSeconds: pointer.Int64Ptr(0)}
		if err := t.hostClient.Delete(ctx, pod, opts); client.IgnoreNotFound(err) != nil {
			allErrs = append(allErrs, fmt.Errorf("unable to delete not ready Terraformer pod: %w", err))
		}
		return apimacherrors.NewAggregate(allErrs)
	}

	isPodTerminated := (podPhase == corev1.PodSucceeded || podPhase == corev1.PodFailed) && len(containerStatuses) > 0
	if containerStateTerminated := containerStatuses[0].State.Terminated; containerStateTerminated != nil && isPodTerminated {
		exitCode := containerStateTerminated.ExitCode

		// Get the logs of the Terraformer pod idependently of the command.
		if err := tfr.GetLogsAndDeletePod(ctx, pod, command, exitCode); err != nil {
			return lsv1alpha1helper.NewWrappedError(err,
				currOp, "GetLogsFromTerraformerPod", fmt.Sprintf("failed to get logs from the Terraformer pod: %s", err.Error()))
		}
	}

	if !isPodTerminated {
		t.log.V(3).Info("Terraformer pod is still running, waiting...")
		return nil
	}

	// Update the provider status only when we have applied a new configuration.
	if command == terraformer.ApplyCommand {
		output, err := tfr.GetOutputFromState(ctx)
		if err != nil {
			err := fmt.Errorf("unable to get terraform output from state: %w", err)
			return lsv1alpha1helper.NewWrappedError(err,
				currOp, "GetOutputFromState", err.Error())
		}

		encStatus, err := encodeProviderStatus(output)
		if err != nil {
			err := fmt.Errorf("unable to update provider status: %w", err)
			return lsv1alpha1helper.NewWrappedError(err,
				currOp, "UpdateProviderStatus", err.Error())
		}

		t.DeployItem.Status.LastError = nil
		t.DeployItem.Status.ProviderStatus = encStatus
		t.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
		// Get deployed generation from pod label.
		appliedItemGeneration, _ := strconv.Atoi(pod.ObjectMeta.Labels[terraformer.LabelKeyGeneration])
		t.DeployItem.Status.ObservedGeneration = int64(appliedItemGeneration)
	}

	// The Terraformer pod destroyed successfully, just clean up is required.
	if command == terraformer.DestroyCommand {
		if err := tfr.EnsureCleanedUp(ctx); err != nil {
			return lsv1alpha1helper.NewWrappedError(err,
				currOp, "EnsureCleanedUp", err.Error())
		}

		controllerutil.RemoveFinalizer(t.DeployItem, lsv1alpha1.LandscaperFinalizer)
		if err := t.lsClient.Update(ctx, t.DeployItem); err != nil {
			return lsv1alpha1helper.NewWrappedError(err,
				currOp, "UpdateItem", err.Error())
		}
	}

	return nil
}

func isPodReady(conditions []corev1.PodCondition) bool {
	if len(conditions) < 0 {
		return false
	}
	for _, condition := range conditions {
		if condition.Type == corev1.PodReady {
			if condition.Status == corev1.ConditionTrue {
				return true
			}
		}
	}
	return false
}

// DeployReconcilePod ensures the terraform configuration and RBAC are up-to-date
// before creating a new Terraformer pod and wait for its creation.
func (t *Terraform) DeployReconcilePod(ctx context.Context, currOp, command string, tfr *terraformer.Terraformer, phase lsv1alpha1.ExecutionPhase) error {

	if err := tfr.EnsureConfig(ctx,
		t.ProviderConfiguration.Main,
		t.ProviderConfiguration.Variables,
		t.ProviderConfiguration.TFVars,
		t.DeployItem.Spec.Configuration.Raw); err != nil {
		return fmt.Errorf("unable to create terraform config: %w", err)
	}

	if err := tfr.EnsureRBAC(ctx); err != nil {
		return fmt.Errorf("unable to create the terraform pod RBAC: %w", err)
	}

	if _, err := tfr.EnsurePod(ctx, command, t.DeployItem.Generation); err != nil {
		return fmt.Errorf("unable to create the Terraformer pod: %w", err)
	}

	t.DeployItem.Status.Phase = phase
	return nil
}

// encodeProviderStatus encodes a terraform provider status to a RawExtension.
func encodeProviderStatus(output json.RawMessage) (*runtime.RawExtension, error) {
	var status = &terraformv1alpha1.ProviderStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: terraformv1alpha1.SchemeGroupVersion.String(),
			Kind:       "ProviderStatus",
		},
	}
	status.Output = output
	raw := &runtime.RawExtension{}
	obj := status.DeepCopyObject()
	if err := runtime.Convert_runtime_Object_To_runtime_RawExtension(&obj, raw, nil); err != nil {
		return &runtime.RawExtension{}, err
	}

	return raw, nil
}
