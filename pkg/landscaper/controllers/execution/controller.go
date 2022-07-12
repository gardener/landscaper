// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"context"
	"fmt"
	"reflect"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/landscaper/pkg/utils"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/gardener/landscaper/controller-utils/pkg/kubernetes"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lserrors "github.com/gardener/landscaper/apis/errors"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/execution"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

// NewController creates a new execution controller that reconcile Execution resources.
func NewController(log logr.Logger, kubeClient client.Client, scheme *runtime.Scheme, eventRecorder record.EventRecorder) (reconcile.Reconciler, error) {
	return &controller{
		log:           log,
		client:        kubeClient,
		scheme:        scheme,
		eventRecorder: eventRecorder,
	}, nil
}

type controller struct {
	log           logr.Logger
	client        client.Client
	eventRecorder record.EventRecorder
	scheme        *runtime.Scheme
}

func (c *controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	if !utils.IsNewReconcile() {
		return c.reconcileOld(ctx, req)
	} else {
		return c.reconcileNew(ctx, req)
	}
}

func (c *controller) reconcileNew(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := c.log.WithValues("resource", req.NamespacedName)
	logger.V(5).Info("reconcile")

	exec := &lsv1alpha1.Execution{}
	if err := read_write_layer.GetExecution(ctx, c.client, req.NamespacedName, exec); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if exec.DeletionTimestamp.IsZero() && !kubernetes.HasFinalizer(exec, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(exec, lsv1alpha1.LandscaperFinalizer)
		if err := c.Writer().UpdateExecution(ctx, read_write_layer.W000086, exec); err != nil {
			return reconcile.Result{}, err
		}
	}

	if lsv1alpha1helper.HasOperation(exec.ObjectMeta, lsv1alpha1.InterruptOperation) {
		if err := c.handleInterruptOperation(ctx, logger, exec); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if exec.Status.JobID != exec.Status.JobIDFinished {
		// Execution is unfinished

		err := c.handleReconcilePhase(ctx, logger, exec)
		return reconcile.Result{}, err
	} else {
		// Execution is finished; nothing to do
		return reconcile.Result{}, nil
	}
}

func (c *controller) handleReconcilePhase(ctx context.Context, log logr.Logger, exec *lsv1alpha1.Execution) lserrors.LsError {
	op := "handleReconcilePhase"

	// A final or empty execution phase means that the current job was not yet started.
	// Switch to start phase Init / InitDelete.
	if exec.Status.ExecutionPhase == lsv1alpha1.ExecPhaseSucceeded ||
		exec.Status.ExecutionPhase == lsv1alpha1.ExecPhaseFailed ||
		exec.Status.ExecutionPhase == lsv1alpha1.ExecPhaseDeleteFailed ||
		exec.Status.ExecutionPhase == "" {

		if exec.DeletionTimestamp.IsZero() {
			exec.Status.ExecutionPhase = lsv1alpha1.ExecPhaseInit
		} else {
			exec.Status.ExecutionPhase = lsv1alpha1.ExecPhaseInitDelete
		}

		// do not use setExecutionPhaseAndUpdate because jobIDFinished should not be set here
		if err := c.Writer().UpdateExecutionStatus(ctx, read_write_layer.W000105, exec); err != nil {
			return lserrors.NewWrappedError(err, op, "UpdateExecutionStatus", err.Error())
		}
	}

	if exec.Status.ExecutionPhase == lsv1alpha1.ExecPhaseInit {
		if err := c.handlePhaseInit(ctx, log, exec); err != nil {
			return c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecPhaseFailed, err)
		}

		if err := c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecPhaseProgressing, nil); err != nil {
			return err
		}
	}

	if exec.Status.ExecutionPhase == lsv1alpha1.ExecPhaseProgressing {
		deployItemClassification, err := c.handlePhaseProgressing(ctx, log, exec)
		if err != nil {
			return c.setExecutionPhaseAndUpdate(ctx, exec, exec.Status.ExecutionPhase, err)
		}

		if !deployItemClassification.HasRunningItems() && deployItemClassification.HasFailedItems() {
			err = lserrors.NewError(op, "handlePhaseProgressing", "failed sub objects")
			return c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecPhaseFailed, err)
		} else if !deployItemClassification.HasRunningItems() && !deployItemClassification.HasRunnableItems() && deployItemClassification.HasPendingItems() {
			err = lserrors.NewError(op, "handlePhaseProgressing", "items could not be started")
			return c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecPhaseFailed, err)
		} else if !deployItemClassification.AllSucceeded() {
			// remain in progressing in all other cases
			err = lserrors.NewError(op, "handlePhaseProgressing", "some running items")
			return c.setExecutionPhaseAndUpdate(ctx, exec, exec.Status.ExecutionPhase, err)
		} else {
			// all succeeded; go to next phase
			if err := c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecPhaseCompleting, nil); err != nil {
				return err
			}
		}
	}

	if exec.Status.ExecutionPhase == lsv1alpha1.ExecPhaseCompleting {
		if err := c.handlePhaseCompleting(ctx, log, exec); err != nil {
			return c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecPhaseFailed, err)
		}

		if err := c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecPhaseSucceeded, nil); err != nil {
			return err
		}
	}

	// Handle deletion phases

	if exec.Status.ExecutionPhase == lsv1alpha1.ExecPhaseInitDelete {
		if err := c.handlePhaseInitDelete(ctx, log, exec); err != nil {
			return c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecPhaseDeleteFailed, err)
		}

		if err := c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecPhaseDeleting, nil); err != nil {
			return err
		}
	}

	if exec.Status.ExecutionPhase == lsv1alpha1.ExecPhaseDeleting {
		deployItemClassification, err := c.handlePhaseDeleting(ctx, log, exec)
		if err != nil {
			return c.setExecutionPhaseAndUpdate(ctx, exec, exec.Status.ExecutionPhase, err)
		}

		if !deployItemClassification.HasRunningItems() && deployItemClassification.HasFailedItems() {
			err = lserrors.NewError(op, "handlePhaseDeleting", "has failed items")
			return c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecPhaseFailed, err)
		} else if !deployItemClassification.HasRunningItems() && !deployItemClassification.HasRunnableItems() && deployItemClassification.HasPendingItems() {
			err = lserrors.NewError(op, "handlePhaseDeleting", "has pending items")
			return c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecPhaseFailed, err)
		}

		// remain in deleting in all other cases,
		// in particular if all deploy items are gone and the finalizer of the execution has been removed.
		if err := c.setExecutionPhaseAndUpdate(ctx, exec, exec.Status.ExecutionPhase, nil); err != nil {
			return err
		}
	}

	return nil
}

func (c *controller) handlePhaseInit(ctx context.Context, log logr.Logger, exec *lsv1alpha1.Execution) lserrors.LsError {
	forceReconcile := false
	o := execution.NewOperation(operation.NewOperation(log, c.client, c.scheme, c.eventRecorder), exec, forceReconcile)

	return o.UpdateDeployItems(ctx)
}

func (c *controller) handlePhaseProgressing(ctx context.Context, log logr.Logger, exec *lsv1alpha1.Execution) (
	*execution.DeployItemClassification, lserrors.LsError) {

	forceReconcile := false
	o := execution.NewOperation(operation.NewOperation(log, c.client, c.scheme, c.eventRecorder), exec, forceReconcile)

	return o.TriggerDeployItems(ctx)
}

func (c *controller) handlePhaseCompleting(ctx context.Context, log logr.Logger, exec *lsv1alpha1.Execution) lserrors.LsError {
	forceReconcile := false
	o := execution.NewOperation(operation.NewOperation(log, c.client, c.scheme, c.eventRecorder), exec, forceReconcile)

	return o.CollectAndUpdateExportsNew(ctx)
}

func (c *controller) handlePhaseInitDelete(ctx context.Context, log logr.Logger, exec *lsv1alpha1.Execution) lserrors.LsError {
	op := "handlePhaseInitDelete"

	forceReconcile := false
	o := execution.NewOperation(operation.NewOperation(log, c.client, c.scheme, c.eventRecorder), exec, forceReconcile)

	managedItems, err := o.ListManagedDeployItems(ctx)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "ListDeployItems", err.Error())
	}

	for i := range managedItems {
		item := &managedItems[i]

		if lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(exec.ObjectMeta) &&
			!lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(item.ObjectMeta) {
			metav1.SetMetaDataAnnotation(&item.ObjectMeta, lsv1alpha1.DeleteWithoutUninstallAnnotation, "true")
			if err := c.Writer().UpdateDeployItem(ctx, read_write_layer.W000104, item); err != nil {
				return lserrors.NewWrappedError(err, "DeleteDeployItem",
					fmt.Sprintf("unable to set deleteWithoutUninstall annotation before deleting deploy item %s / %s", item.Namespace, item.Name), err.Error())
			}
		}

		if item.DeletionTimestamp.IsZero() {
			if err := o.Writer().DeleteDeployItem(ctx, read_write_layer.W000112, item); client.IgnoreNotFound(err) != nil {
				return lserrors.NewWrappedError(err, "DeleteDeployItem",
					fmt.Sprintf("unable to delete deploy item %s / %s", item.Namespace, item.Name), err.Error())
			}
		}
	}

	return nil
}

func (c *controller) handlePhaseDeleting(ctx context.Context, log logr.Logger, exec *lsv1alpha1.Execution) (
	*execution.DeployItemClassification, lserrors.LsError) {

	forceReconcile := false
	o := execution.NewOperation(operation.NewOperation(log, c.client, c.scheme, c.eventRecorder), exec, forceReconcile)

	return o.TriggerDeployItemsForDelete(ctx)
}

func (c *controller) reconcileOld(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := c.log.WithValues("resource", req.NamespacedName)
	logger.V(5).Info("reconcile")

	exec := &lsv1alpha1.Execution{}
	if err := read_write_layer.GetExecution(ctx, c.client, req.NamespacedName, exec); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// don't reconcile if ignore annotation is set and execution is not currently running
	if lsv1alpha1helper.HasIgnoreAnnotation(exec.ObjectMeta) && lsv1alpha1helper.IsCompletedExecutionPhase(exec.Status.Phase) {
		logger.V(7).Info("skipping reconcile due to ignore annotation")
		return reconcile.Result{}, nil
	}

	oldExec := exec.DeepCopy()

	lsError := c.Ensure(ctx, logger, exec)

	lsErr2 := c.removeForceReconcileAnnotation(ctx, exec)
	if lsError == nil {
		// lsError is more important than lsErr2
		lsError = lsErr2
	}

	isDelete := !exec.DeletionTimestamp.IsZero()
	return reconcile.Result{}, handleError(ctx, lsError, logger, c.client, c.eventRecorder, oldExec, exec, isDelete)
}

func (c *controller) Ensure(ctx context.Context, log logr.Logger, exec *lsv1alpha1.Execution) lserrors.LsError {
	if err := HandleAnnotationsAndGeneration(ctx, log, c.client, exec); err != nil {
		return err
	}

	if exec.DeletionTimestamp.IsZero() && !kubernetes.HasFinalizer(exec, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(exec, lsv1alpha1.LandscaperFinalizer)
		if err := c.Writer().UpdateExecution(ctx, read_write_layer.W000025, exec); err != nil {
			return lserrors.NewError("Reconcile", "AddFinalizer", err.Error())
		}
	}

	forceReconcile := lsv1alpha1helper.HasOperation(exec.ObjectMeta, lsv1alpha1.ForceReconcileOperation)
	op := execution.NewOperation(operation.NewOperation(log, c.client, c.scheme, c.eventRecorder), exec,
		forceReconcile)

	if !exec.DeletionTimestamp.IsZero() {
		return op.Delete(ctx)
	}

	if lsv1alpha1helper.IsCompletedExecutionPhase(exec.Status.Phase) {
		err := op.HandleDeployItemPhaseAndGenerationChanges(ctx, log)
		if err != nil {
			return lserrors.NewWrappedError(err, "Reconcile", "HandleDeployItemPhaseAndGenerationChanges", err.Error())
		}
		if lsv1alpha1helper.IsCompletedExecutionPhase(exec.Status.Phase) {
			return nil
		}
	}

	return op.Reconcile(ctx)
}

func (c *controller) removeForceReconcileAnnotation(ctx context.Context, exec *lsv1alpha1.Execution) lserrors.LsError {
	if lsv1alpha1helper.HasOperation(exec.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
		old := exec.DeepCopy()
		delete(exec.Annotations, lsv1alpha1.OperationAnnotation)
		writer := read_write_layer.NewWriter(c.log, c.client)
		if err := writer.PatchExecution(ctx, read_write_layer.W000029, exec, old); err != nil {
			c.eventRecorder.Event(exec, corev1.EventTypeWarning, "RemoveForceReconcileAnnotation", err.Error())
			return lserrors.NewWrappedError(err, "Reconcile", "RemoveForceReconcileAnnotation", err.Error())
		}
	}
	return nil
}

func (c *controller) Writer() *read_write_layer.Writer {
	return read_write_layer.NewWriter(c.log, c.client)
}

func (c *controller) handleInterruptOperation(ctx context.Context, log logr.Logger, exec *lsv1alpha1.Execution) error {
	delete(exec.Annotations, lsv1alpha1.OperationAnnotation)
	if err := c.Writer().UpdateExecution(ctx, read_write_layer.W000100, exec); err != nil {
		return err
	}

	op := "handleInterruptOperation"

	forceReconcile := false
	o := execution.NewOperation(operation.NewOperation(log, c.client, c.scheme, c.eventRecorder), exec, forceReconcile)

	managedItems, err := o.ListManagedDeployItems(ctx)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "ListDeployItems", err.Error())
	}

	for i := range managedItems {
		item := &managedItems[i]

		if item.Status.JobIDFinished != exec.Status.JobID {
			item.Status.JobID = exec.Status.JobID
			item.Status.JobIDFinished = exec.Status.JobID
			item.Status.DeployItemPhase = lsv1alpha1.DeployItemPhaseFailed
			item.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
			item.Status.LastError = lserrors.UpdatedError(item.Status.LastError,
				"InterruptOperation",
				"InterruptOperation",
				"operation was interrupted")

			if err := o.Writer().UpdateDeployItemStatus(ctx, read_write_layer.W000101, item); err != nil {
				return lserrors.NewWrappedError(err, "UpdateDeployItemStatus",
					fmt.Sprintf("unable to update deploy item %s / %s for interrupt", item.Namespace, item.Name), err.Error())
			}
		}
	}

	return nil
}

func (c *controller) setExecutionPhaseAndUpdate(ctx context.Context, exec *lsv1alpha1.Execution, phase lsv1alpha1.ExecPhase, lsErr lserrors.LsError) lserrors.LsError {
	exec.Status.LastError = lserrors.TryUpdateLsError(exec.Status.LastError, lsErr)

	if lsErr != nil {
		c.log.Error(lsErr, "setExecutionPhaseAndUpdate:"+lsErr.Error())
	}

	exec.Status.ExecutionPhase = phase

	if exec.Status.ExecutionPhase == lsv1alpha1.ExecPhaseSucceeded ||
		exec.Status.ExecutionPhase == lsv1alpha1.ExecPhaseFailed ||
		exec.Status.ExecutionPhase == lsv1alpha1.ExecPhaseDeleteFailed {

		exec.Status.JobIDFinished = exec.Status.JobID
	}

	if err := c.Writer().UpdateExecutionStatus(ctx, read_write_layer.W000105, exec); err != nil {

		if exec.Status.ExecutionPhase == lsv1alpha1.ExecPhaseDeleting {
			// recheck if already deleted
			execRecheck := &lsv1alpha1.Execution{}
			errRecheck := read_write_layer.GetExecution(ctx, c.client, kutil.ObjectKey(exec.Name, exec.Namespace), execRecheck)
			if errRecheck != nil && apierrors.IsNotFound(errRecheck) {
				return nil
			}
		}

		c.log.Error(err, "unable to update status")

		if lsErr == nil {
			return lserrors.NewWrappedError(err, "setExecutionPhaseAndUpdate", "UpdateExecutionStatus", err.Error())
		}
	}

	return lsErr
}

// HandleAnnotationsAndGeneration is meant to be called at the beginning of the reconcile loop.
// If a reconcile is needed due to the reconcile annotation or a change in the generation, it will set the phase to Init and remove the reconcile annotation.
// Returns: an error, if updating the execution failed, nil otherwise
func HandleAnnotationsAndGeneration(ctx context.Context, log logr.Logger, c client.Client, exec *lsv1alpha1.Execution) lserrors.LsError {
	operation := "HandleAnnotationsAndGeneration"
	hasReconcileAnnotation := lsv1alpha1helper.HasOperation(exec.ObjectMeta, lsv1alpha1.ReconcileOperation)
	hasForceReconcileAnnotation := lsv1alpha1helper.HasOperation(exec.ObjectMeta, lsv1alpha1.ForceReconcileOperation)
	if hasReconcileAnnotation || hasForceReconcileAnnotation || exec.Status.ObservedGeneration != exec.Generation {
		// reconcile necessary due to one of
		// - reconcile annotation
		// - force-reconcile annotation
		// - outdated generation
		opAnn := lsv1alpha1helper.GetOperation(exec.ObjectMeta)
		log.V(5).Info("reconcile required, setting observed generation and phase", "operationAnnotation", opAnn, "observedGeneration", exec.Status.ObservedGeneration, "generation", exec.Generation)
		exec.Status.ObservedGeneration = exec.Generation
		exec.Status.Phase = lsv1alpha1.ExecutionPhaseInit

		log.V(7).Info("updating status")
		writer := read_write_layer.NewWriter(log, c)
		if err := writer.UpdateExecutionStatus(ctx, read_write_layer.W000033, exec); err != nil {
			return lserrors.NewWrappedError(err, operation, "update execution status", err.Error())
		}
		log.V(7).Info("successfully updated status")
	}
	if hasReconcileAnnotation {
		log.V(5).Info("removing reconcile annotation")
		delete(exec.ObjectMeta.Annotations, lsv1alpha1.OperationAnnotation)
		log.V(7).Info("updating metadata")
		writer := read_write_layer.NewWriter(log, c)
		if err := writer.UpdateExecution(ctx, read_write_layer.W000027, exec); err != nil {
			return lserrors.NewWrappedError(err, operation, "update execution", err.Error())
		}
		log.V(7).Info("successfully updated metadata")
	}

	return nil
}

func handleError(ctx context.Context, err lserrors.LsError, log logr.Logger, c client.Client,
	eventRecorder record.EventRecorder, oldExec, exec *lsv1alpha1.Execution, isDelete bool) error {
	// if successfully deleted we could not update the object
	if isDelete && err == nil {
		exec2 := &lsv1alpha1.Execution{}
		if err2 := read_write_layer.GetExecution(ctx, c, kutil.ObjectKey(exec.Name, exec.Namespace), exec2); err2 != nil {
			if apierrors.IsNotFound(err2) {
				return nil
			}
		}
	}

	// There are two kind of errors: err != nil and exec.Status.LastError != nil
	// If err != nil this error is set and returned such that a retry is initiated.
	// If err == nil and exec.Status.LastError != nil another object must change its state and initiate a new event
	// for the execution exec.
	if err != nil {
		log.Error(err, "handleError")
		exec.Status.LastError = lserrors.TryUpdateLsError(exec.Status.LastError, err)
	}

	exec.Status.Phase = lsv1alpha1.ExecutionPhase(lserrors.GetPhaseForLastError(
		lsv1alpha1.ComponentInstallationPhase(exec.Status.Phase),
		exec.Status.LastError,
		5*time.Minute),
	)

	if exec.Status.LastError != nil {
		lastErr := exec.Status.LastError
		eventRecorder.Event(exec, corev1.EventTypeWarning, lastErr.Reason, lastErr.Message)
	}

	if !reflect.DeepEqual(oldExec.Status, exec.Status) {
		writer := read_write_layer.NewWriter(log, c)
		if updateErr := writer.UpdateExecutionStatus(ctx, read_write_layer.W000031, exec); updateErr != nil {
			if apierrors.IsConflict(updateErr) { // reduce logging
				log.V(5).Info(fmt.Sprintf("unable to update status: %s", updateErr.Error()))
			} else {
				log.Error(updateErr, "unable to update status")
			}
			return updateErr
		}
	}
	return err
}
