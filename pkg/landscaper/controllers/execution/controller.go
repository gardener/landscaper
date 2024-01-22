// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lserrors "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/landscaper/execution"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	lsutil "github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/lock"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

// NewController creates a new execution controller that reconcile Execution resources.
func NewController(lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient client.Client,
	logger logging.Logger, scheme *runtime.Scheme, eventRecorder record.EventRecorder, maxNumberOfWorker int,
	lockingEnabled bool, callerName string) (reconcile.Reconciler, error) {

	ctx := logging.NewContext(context.Background(), logger)

	wc := lsutil.NewWorkerCounter(maxNumberOfWorker)

	finishedObjectCache, err := prepareFinishedObjectCache(ctx, lsUncachedClient)
	if err != nil {
		return nil, err
	}

	return &controller{
		lsUncachedClient:    lsUncachedClient,
		lsCachedClient:      lsCachedClient,
		hostUncachedClient:  hostUncachedClient,
		hostCachedClient:    hostCachedClient,
		finishedObjectCache: finishedObjectCache,
		log:                 logger,
		scheme:              scheme,
		eventRecorder:       eventRecorder,
		workerCounter:       wc,
		lockingEnabled:      lockingEnabled,
		callerName:          callerName,
		locker:              *lock.NewLocker(lsUncachedClient, hostUncachedClient, callerName),
	}, nil
}

type controller struct {
	lsUncachedClient    client.Client
	lsCachedClient      client.Client
	hostUncachedClient  client.Client
	hostCachedClient    client.Client
	finishedObjectCache *lsutil.FinishedObjectCache

	log            logging.Logger
	eventRecorder  record.EventRecorder
	scheme         *runtime.Scheme
	workerCounter  *lsutil.WorkerCounter
	lockingEnabled bool
	callerName     string
	locker         lock.Locker
}

func prepareFinishedObjectCache(ctx context.Context, lsUncachedClient client.Client) (*lsutil.FinishedObjectCache, error) {
	log, ctx := logging.FromContextOrNew(ctx, nil)

	finishedObjectCache := lsutil.NewFinishedObjectCache()
	namespaces := &corev1.NamespaceList{}
	if err := read_write_layer.ListNamespaces(ctx, lsUncachedClient, namespaces, read_write_layer.R000099); err != nil {
		return nil, err
	}

	perfTotal := lsutil.StartPerformanceMeasurement(&log, "prepare finished object for executions")
	defer perfTotal.Stop()

	for _, namespace := range namespaces.Items {
		perf := lsutil.StartPerformanceMeasurement(&log, "prepare finished object cache for executions: fetch from namespace "+namespace.Name)

		execList := &lsv1alpha1.ExecutionList{}
		if err := read_write_layer.ListExecutions(ctx, lsUncachedClient, execList, read_write_layer.R000100,
			client.InNamespace(namespace.Name)); err != nil {
			return nil, err
		}

		perf.Stop()

		perf = lsutil.StartPerformanceMeasurement(&log, "prepare finished object cache for executions: add for namespace "+namespace.Name)

		for execIndex := range execList.Items {
			exec := &execList.Items[execIndex]
			if isExecFinished(exec) {
				finishedObjectCache.Add(&exec.ObjectMeta)
			}
		}

		perf.Stop()
	}

	return finishedObjectCache, nil
}

func (c *controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := c.log.StartReconcile(req)
	ctx = logging.NewContext(ctx, logger)

	c.workerCounter.EnterWithLog(logger, 70, "executions")
	defer c.workerCounter.Exit()

	if c.finishedObjectCache.IsContained(req) {
		cachedMetadata := lsutil.EmptyExecutionMetadata()
		if err := read_write_layer.GetMetaData(ctx, c.lsCachedClient, req.NamespacedName, cachedMetadata, read_write_layer.R000101); err != nil {
			if apierrors.IsNotFound(err) {
				logger.Debug(err.Error())
				return reconcile.Result{}, nil
			}
			return lsutil.LogHelper{}.LogStandardErrorAndGetReconcileResult(ctx, err)
		}

		if c.finishedObjectCache.IsFinishedAndRemove(cachedMetadata) {
			return reconcile.Result{}, nil
		}
	}

	if c.lockingEnabled {
		metadata := lsutil.EmptyExecutionMetadata()
		if err := c.lsUncachedClient.Get(ctx, req.NamespacedName, metadata); err != nil {
			if apierrors.IsNotFound(err) {
				logger.Debug(err.Error())
				return reconcile.Result{}, nil
			}
			return lsutil.LogHelper{}.LogStandardErrorAndGetReconcileResult(ctx, err)
		}

		syncObject, err := c.locker.LockExecution(ctx, metadata)
		if err != nil {
			return lsutil.LogHelper{}.LogErrorAndGetReconcileResult(ctx, err)
		}

		if syncObject == nil {
			return c.locker.NotLockedResult()
		}

		defer func() {
			c.locker.Unlock(ctx, syncObject)
		}()
	}

	exec := &lsv1alpha1.Execution{}
	if err := read_write_layer.GetExecution(ctx, c.lsUncachedClient, req.NamespacedName, exec, read_write_layer.R000019); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info(err.Error())
			return reconcile.Result{}, nil
		}
		return lsutil.LogHelper{}.LogStandardErrorAndGetReconcileResult(ctx, err)
	}

	if needsFinalizer(exec) {
		controllerutil.AddFinalizer(exec, lsv1alpha1.LandscaperFinalizer)
		if err := c.Writer().UpdateExecution(ctx, read_write_layer.W000086, exec); err != nil {
			return lsutil.LogHelper{}.LogStandardErrorAndGetReconcileResult(ctx, err)
		}
	}

	if hasInterruptOperation(exec) {
		if err := c.handleInterruptOperation(ctx, exec); err != nil {
			return lsutil.LogHelper{}.LogStandardErrorAndGetReconcileResult(ctx, err)
		}
		return reconcile.Result{}, nil
	}

	if isDifferentJobIDs(exec) {
		// Execution is unfinished

		err := c.handleReconcilePhase(ctx, exec)
		return lsutil.LogHelper{}.LogErrorAndGetReconcileResult(ctx, err)
	} else {
		// Execution is finished; nothing to do
		return reconcile.Result{}, nil
	}
}

func (c *controller) handleReconcilePhase(ctx context.Context, exec *lsv1alpha1.Execution) lserrors.LsError {

	op := "handleReconcilePhase"

	// the cache is only available and reliable if not nil
	var deployItemCache = exec.Status.DeployItemCache

	// A final or empty execution phase means that the current job was not yet started.
	// Switch to start phase Init / InitDelete.
	if exec.Status.ExecutionPhase.IsFinal() || exec.Status.ExecutionPhase.IsEmpty() {

		if exec.Status.ExecutionPhase.IsEmpty() {
			deployItemCache = &lsv1alpha1.DeployItemCache{}
		}

		exec.Status.DeployItemCache = nil

		if exec.DeletionTimestamp.IsZero() {
			exec.Status.ExecutionPhase = lsv1alpha1.ExecutionPhases.Init
		} else {
			exec.Status.ExecutionPhase = lsv1alpha1.ExecutionPhases.InitDelete
		}

		now := metav1.Now()
		exec.Status.PhaseTransitionTime = &now

		exec.Status.TransitionTimes = lsutil.SetInitTransitionTime(exec.Status.TransitionTimes)

		// do not use setExecutionPhaseAndUpdate because jobIDFinished should not be set here
		if err := c.Writer().UpdateExecutionStatus(ctx, read_write_layer.W000105, exec); err != nil {
			return lserrors.NewWrappedError(err, op, "UpdateExecutionStatus", err.Error())
		}
	}

	if exec.Status.ExecutionPhase == lsv1alpha1.ExecutionPhases.Init {
		if err := c.handlePhaseInit(ctx, exec, deployItemCache); err != nil {
			if lsutil.IsRecoverableError(err) {
				return c.setExecutionPhaseAndUpdate(ctx, exec, exec.Status.ExecutionPhase, err, read_write_layer.W000007)
			}
			return c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecutionPhases.Failed, err, read_write_layer.W000131)
		}

		exec.Status.TransitionTimes = lsutil.SetWaitTransitionTime(exec.Status.TransitionTimes)

		if err := c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecutionPhases.Progressing, nil, read_write_layer.W000132); err != nil {
			return err
		}
	}

	if exec.Status.ExecutionPhase == lsv1alpha1.ExecutionPhases.Progressing {
		deployItemClassification, err := c.handlePhaseProgressing(ctx, exec)
		if err != nil {
			return c.setExecutionPhaseAndUpdate(ctx, exec, exec.Status.ExecutionPhase, err, read_write_layer.W000133)
		}

		if !deployItemClassification.HasRunningItems() && deployItemClassification.HasFailedItems() {
			err = lserrors.NewError(op, "handlePhaseProgressing", "has failed or missing deploy items", lsv1alpha1.ErrorForInfoOnly)
			return c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecutionPhases.Failed, err, read_write_layer.W000134)
		} else if !deployItemClassification.HasRunningItems() && !deployItemClassification.HasRunnableItems() && deployItemClassification.HasPendingItems() {
			err = lserrors.NewError(op, "handlePhaseProgressing", "items could not be started", lsv1alpha1.ErrorForInfoOnly)
			return c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecutionPhases.Failed, err, read_write_layer.W000135)
		} else if !deployItemClassification.AllSucceeded() {
			// remain in progressing in all other cases
			err = lserrors.NewError(op, "handlePhaseProgressing", "some running items", lsv1alpha1.ErrorUnfinished,
				lsv1alpha1.ErrorForInfoOnly, lsv1alpha1.ErrorNoRetry)
			return c.setExecutionPhaseAndUpdate(ctx, exec, exec.Status.ExecutionPhase, err, read_write_layer.W000136)
		} else {
			// all succeeded; go to next phase
			if err := c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecutionPhases.Completing, nil, read_write_layer.W000137); err != nil {
				return err
			}
		}
	}

	if exec.Status.ExecutionPhase == lsv1alpha1.ExecutionPhases.Completing {
		if err := c.handlePhaseCompleting(ctx, exec); err != nil {
			if lsutil.IsRecoverableError(err) {
				return c.setExecutionPhaseAndUpdate(ctx, exec, exec.Status.ExecutionPhase, err, read_write_layer.W000008)
			}
			return c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecutionPhases.Failed, err, read_write_layer.W000138)
		}

		if err := c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecutionPhases.Succeeded, nil, read_write_layer.W000139); err != nil {
			return err
		}
	}

	// Handle deletion phases

	if exec.Status.ExecutionPhase == lsv1alpha1.ExecutionPhases.InitDelete {
		if err := c.handlePhaseInitDelete(ctx, exec); err != nil {
			if lsutil.IsRecoverableError(err) {
				return c.setExecutionPhaseAndUpdate(ctx, exec, exec.Status.ExecutionPhase, err, read_write_layer.W000010)
			}
			return c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecutionPhases.DeleteFailed, err, read_write_layer.W000140)
		}

		if err := c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecutionPhases.Deleting, nil, read_write_layer.W000141); err != nil {
			return err
		}
	}

	if exec.Status.ExecutionPhase == lsv1alpha1.ExecutionPhases.Deleting {
		deployItemClassification, err := c.handlePhaseDeleting(ctx, exec)
		if err != nil {
			return c.setExecutionPhaseAndUpdate(ctx, exec, exec.Status.ExecutionPhase, err, read_write_layer.W000142)
		}

		if !deployItemClassification.HasRunningItems() && deployItemClassification.HasFailedItems() {
			err = lserrors.NewError(op, "handlePhaseDeleting", "has failed items", lsv1alpha1.ErrorForInfoOnly)
			return c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecutionPhases.DeleteFailed, err, read_write_layer.W000143)
		} else if !deployItemClassification.HasRunningItems() && !deployItemClassification.HasRunnableItems() && deployItemClassification.HasPendingItems() {
			err = lserrors.NewError(op, "handlePhaseDeleting", "has pending items", lsv1alpha1.ErrorForInfoOnly)
			return c.setExecutionPhaseAndUpdate(ctx, exec, lsv1alpha1.ExecutionPhases.DeleteFailed, err, read_write_layer.W000144)
		}

		// remain in deleting in all other cases,
		// in particular if all deploy items are gone and the finalizer of the execution has been removed.
		if err := c.setExecutionPhaseAndUpdate(ctx, exec, exec.Status.ExecutionPhase, nil, read_write_layer.W000145); err != nil {
			return err
		}
	}

	return nil
}

func (c *controller) handlePhaseInit(ctx context.Context, exec *lsv1alpha1.Execution, deployItemCache *lsv1alpha1.DeployItemCache) lserrors.LsError {
	forceReconcile := false
	o := execution.NewOperation(operation.NewOperation(c.scheme, c.eventRecorder, c.lsUncachedClient), exec, forceReconcile)

	return o.UpdateDeployItems(ctx, deployItemCache)
}

func (c *controller) handlePhaseProgressing(ctx context.Context, exec *lsv1alpha1.Execution) (
	*execution.DeployItemClassification, lserrors.LsError) {
	forceReconcile := false
	o := execution.NewOperation(operation.NewOperation(c.scheme, c.eventRecorder, c.lsUncachedClient), exec, forceReconcile)

	return o.TriggerDeployItems(ctx)
}

func (c *controller) handlePhaseCompleting(ctx context.Context, exec *lsv1alpha1.Execution) lserrors.LsError {

	if exec.Status.DeployItemCache != nil {
		exec.Status.DeployItemCache.OrphanedDIs = nil
	}

	forceReconcile := false
	o := execution.NewOperation(operation.NewOperation(c.scheme, c.eventRecorder, c.lsUncachedClient), exec, forceReconcile)

	return o.CollectAndUpdateExportsNew(ctx)
}

func (c *controller) handlePhaseInitDelete(ctx context.Context, exec *lsv1alpha1.Execution) lserrors.LsError {
	op := "handlePhaseInitDelete"

	forceReconcile := false
	o := execution.NewOperation(operation.NewOperation(c.scheme, c.eventRecorder, c.lsUncachedClient), exec, forceReconcile)

	managedItems, err := o.ListManagedDeployItems(ctx, read_write_layer.R000083, exec.Status.DeployItemCache)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "ListDeployItems", err.Error())
	}

	for i := range managedItems {
		item := managedItems[i]

		if lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(exec.ObjectMeta) &&
			!lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(item.ObjectMeta) {
			metav1.SetMetaDataAnnotation(&item.ObjectMeta, lsv1alpha1.DeleteWithoutUninstallAnnotation, "true")
			if err := c.Writer().UpdateDeployItem(ctx, read_write_layer.W000104, item); err != nil {
				return lserrors.NewWrappedError(err, "DeleteDeployItem",
					fmt.Sprintf("unable to set deleteWithoutUninstall annotation before deleting deploy item %s / %s", item.Namespace, item.Name), err.Error())
			}
		}

		if item.DeletionTimestamp.IsZero() {
			if err := o.WriterToLsUncachedClient().DeleteDeployItem(ctx, read_write_layer.W000112, item); client.IgnoreNotFound(err) != nil {
				return lserrors.NewWrappedError(err, "DeleteDeployItem",
					fmt.Sprintf("unable to delete deploy item %s / %s", item.Namespace, item.Name), err.Error())
			}
		}
	}

	return nil
}

func (c *controller) handlePhaseDeleting(ctx context.Context, exec *lsv1alpha1.Execution) (
	*execution.DeployItemClassification, lserrors.LsError) {
	forceReconcile := false
	o := execution.NewOperation(operation.NewOperation(c.scheme, c.eventRecorder, c.lsUncachedClient), exec, forceReconcile)

	return o.TriggerDeployItemsForDelete(ctx)
}

func (c *controller) Writer() *read_write_layer.Writer {
	return read_write_layer.NewWriter(c.lsUncachedClient)
}

func (c *controller) handleInterruptOperation(ctx context.Context, exec *lsv1alpha1.Execution) error {
	delete(exec.Annotations, lsv1alpha1.OperationAnnotation)
	if err := c.Writer().UpdateExecution(ctx, read_write_layer.W000100, exec); err != nil {
		return err
	}

	op := "handleInterruptOperation"

	forceReconcile := false
	o := execution.NewOperation(operation.NewOperation(c.scheme, c.eventRecorder, c.lsUncachedClient), exec, forceReconcile)

	managedItems, err := o.ListManagedDeployItems(ctx, read_write_layer.R000058, nil)
	if err != nil {
		return lserrors.NewWrappedError(err, op, "ListDeployItems", err.Error())
	}

	for i := range managedItems {
		item := managedItems[i]

		if item.Status.JobIDFinished != exec.Status.JobID {
			item.Status.SetJobID(exec.Status.JobID)
			item.Status.JobIDFinished = exec.Status.JobID
			item.Status.TransitionTimes = lsutil.SetFinishedTransitionTime(item.Status.TransitionTimes)
			lsv1alpha1helper.SetDeployItemToFailed(item)
			lsutil.SetLastError(&item.Status, lserrors.UpdatedError(item.Status.GetLastError(),
				"InterruptOperation",
				"InterruptOperation",
				"operation was interrupted"))

			if err := o.WriterToLsUncachedClient().UpdateDeployItemStatus(ctx, read_write_layer.W000101, item); err != nil {
				return lserrors.NewWrappedError(err, "UpdateDeployItemStatus",
					fmt.Sprintf("unable to update deploy item %s / %s for interrupt", item.Namespace, item.Name), err.Error())
			}
		}
	}

	return nil
}

func (c *controller) setExecutionPhaseAndUpdate(ctx context.Context, exec *lsv1alpha1.Execution,
	phase lsv1alpha1.ExecutionPhase, lsErr lserrors.LsError, writeID read_write_layer.WriteID) lserrors.LsError {

	logger, ctx := logging.FromContextOrNew(ctx, nil)

	exec.Status.LastError = lserrors.TryUpdateLsError(exec.Status.LastError, lsErr)

	if phase != exec.Status.ExecutionPhase {
		now := metav1.Now()
		exec.Status.PhaseTransitionTime = &now
	}
	exec.Status.ExecutionPhase = phase

	if exec.Status.ExecutionPhase.IsFinal() {
		exec.Status.JobIDFinished = exec.Status.JobID
		exec.Status.TransitionTimes = lsutil.SetFinishedTransitionTime(exec.Status.TransitionTimes)
	}

	if err := c.Writer().UpdateExecutionStatus(ctx, writeID, exec); err != nil {

		if exec.Status.ExecutionPhase == lsv1alpha1.ExecutionPhases.Deleting {
			// recheck if already deleted
			execRecheck := &lsv1alpha1.Execution{}
			errRecheck := read_write_layer.GetExecution(ctx, c.lsUncachedClient, kutil.ObjectKey(exec.Name, exec.Namespace),
				execRecheck, read_write_layer.R000021)
			if errRecheck != nil && apierrors.IsNotFound(errRecheck) {
				return nil
			}
		}

		logger.Error(err, "unable to update status")

		if lsErr == nil {
			return lserrors.NewWrappedError(err, "setExecutionPhaseAndUpdate", "UpdateExecutionStatus", err.Error())
		}
	}

	return lsErr
}
