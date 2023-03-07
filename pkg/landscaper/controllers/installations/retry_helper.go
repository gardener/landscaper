// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"time"

	"k8s.io/utils/clock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

const (
	reconcileReasonRetry = "retry"
)

var (
	defaultRetryDurationForFailed          = 5 * time.Minute
	defaultRetryDurationForNewAndSucceeded = 24 * time.Hour
)

type retryHelper struct {
	cl     client.Client
	writer *read_write_layer.Writer
	clock  clock.PassiveClock
}

func newRetryHelper(cl client.Client, passiveClock clock.PassiveClock) *retryHelper {
	return &retryHelper{
		cl:     cl,
		writer: read_write_layer.NewWriter(cl),
		clock:  passiveClock,
	}
}

func (r *retryHelper) preProcessRetry(ctx context.Context, inst *lsv1alpha1.Installation) error {

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ReconcileOperation) &&
		!r.hasReconcileReasonRetry(inst.ObjectMeta) {
		// reconcile was not triggered by the retry mechanism, therefore we reset the retry status
		if err := r.resetRetryStatus(ctx, inst); err != nil {
			return err
		}
	}

	return nil
}

// hasReconcileReasonRetry returns true if the annotation "landscaper.gardener.cloud/reconcile-reason" is set and has
// the value "retry". This annotation is set together with a reconcile annotation to mark it as set by the retry
// mechanism (as opposed to reconcile annotations set by a user, for example). This distinction is necessary for the
// reset of the retry status.
func (r *retryHelper) hasReconcileReasonRetry(obj metav1.ObjectMeta) bool {
	reason, found := obj.GetAnnotations()[lsv1alpha1.ReconcileReasonAnnotation]
	return found && reason == reconcileReasonRetry
}

func (r *retryHelper) recomputeRetry(ctx context.Context, inst *lsv1alpha1.Installation, oldResult reconcile.Result, oldError error) (reconcile.Result, error) {

	if metav1.HasAnnotation(inst.ObjectMeta, lsv1alpha1.OperationAnnotation) {
		return oldResult, oldError
	}

	isUpToDate := inst.Status.ObservedGeneration == inst.GetGeneration()
	if !isUpToDate {
		return oldResult, oldError
	}

	if inst.Status.AutomaticReconcileStatus != nil {
		if inst.Status.AutomaticReconcileStatus.Generation != inst.GetGeneration() ||
			(inst.Status.InstallationPhase == lsv1alpha1.InstallationPhases.Succeeded && inst.Status.AutomaticReconcileStatus.OnFailed) ||
			(inst.Status.InstallationPhase.IsFailed() && !inst.Status.AutomaticReconcileStatus.OnFailed) {

			if err := r.resetRetryStatus(ctx, inst); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	if r.isRetryActivatedForFailed(inst) && r.isFailed(inst) {
		return r.recomputeRetryForFailed(ctx, inst, oldResult, oldError)

	} else if r.isRetryActivatedForSucceeded(inst) && r.isSucceeded(inst) {
		return r.recomputeRetryForNewAndSucceeded(ctx, inst)

	} else {
		return oldResult, oldError
	}
}

func (r *retryHelper) isFailed(inst *lsv1alpha1.Installation) bool {
	return inst.Status.InstallationPhase.IsFailed()
}

func (r *retryHelper) isRetryActivatedForFailed(inst *lsv1alpha1.Installation) bool {
	return inst.Spec.AutomaticReconcile != nil && inst.Spec.AutomaticReconcile.FailedReconcile != nil &&
		(inst.Spec.AutomaticReconcile.FailedReconcile.NumberOfReconciles == nil || *inst.Spec.AutomaticReconcile.FailedReconcile.NumberOfReconciles > 0)
}

func (r *retryHelper) recomputeRetryForFailed(ctx context.Context, inst *lsv1alpha1.Installation, oldResult reconcile.Result, oldError error) (reconcile.Result, error) {
	retryStatus := inst.Status.AutomaticReconcileStatus

	// first failure, or installation changed
	if retryStatus == nil {
		if err := r.addReconcileAnnotation(ctx, inst); err != nil {
			return reconcile.Result{}, err
		}

		if err := r.updateRetryStatus(ctx, inst, 1, true); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if r.maxNumberOfRetriesDoneForFailed(inst) {
		return oldResult, oldError
	}

	if r.isNextRetryDueForFailed(inst) {
		if err := r.addReconcileAnnotation(ctx, inst); err != nil {
			return reconcile.Result{}, err
		}

		numEntries := retryStatus.NumberOfReconciles + 1
		if err := r.updateRetryStatus(ctx, inst, numEntries, true); err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	// too early
	return reconcile.Result{
		Requeue:      true,
		RequeueAfter: r.getDurationUntilNextRetryForFailed(inst),
	}, nil
}

func (r *retryHelper) updateRetryStatus(ctx context.Context, inst *lsv1alpha1.Installation, numRetries int, onFailed bool) error {
	logger, ctx := logging.FromContextOrNew(ctx, nil)

	// set inst.Status.RetryFailedStatus
	inst.Status.AutomaticReconcileStatus = &lsv1alpha1.AutomaticReconcileStatus{
		Generation:         inst.GetGeneration(),
		NumberOfReconciles: numRetries,
		LastReconcileTime:  r.metaNow(),
		OnFailed:           onFailed,
	}

	logger.Info("update retry status", "numberOfRetries", numRetries, "onFailed", onFailed)
	if err := r.writer.UpdateInstallationStatus(ctx, read_write_layer.W000028, inst); err != nil {
		logger.Error(err, "failed to update retry status")
		return err
	}

	return nil
}

func (r *retryHelper) resetRetryStatus(ctx context.Context, inst *lsv1alpha1.Installation) error {
	logger, ctx := logging.FromContextOrNew(ctx, nil)

	logger.Info("reset retry status")
	inst.Status.AutomaticReconcileStatus = nil
	if err := r.writer.UpdateInstallationStatus(ctx, read_write_layer.W000027, inst); err != nil {
		logger.Error(err, "failed to reset retry status")
		return err
	}

	return nil
}

func (r *retryHelper) isSucceeded(inst *lsv1alpha1.Installation) bool {
	return inst.Status.InstallationPhase == lsv1alpha1.InstallationPhases.Succeeded
}

func (r *retryHelper) isRetryActivatedForSucceeded(inst *lsv1alpha1.Installation) bool {
	return inst.Spec.AutomaticReconcile != nil && inst.Spec.AutomaticReconcile.SucceededReconcile != nil
}

func (r *retryHelper) recomputeRetryForNewAndSucceeded(ctx context.Context, inst *lsv1alpha1.Installation) (reconcile.Result, error) {
	retryStatus := inst.Status.AutomaticReconcileStatus

	if retryStatus == nil {
		if err := r.updateRetryStatus(ctx, inst, 0, false); err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	if r.isNextRetryDueForNewAndSucceeded(inst) {
		if err := r.addReconcileAnnotation(ctx, inst); err != nil {
			return reconcile.Result{}, err
		}

		numEntries := retryStatus.NumberOfReconciles + 1
		if err := r.updateRetryStatus(ctx, inst, numEntries, false); err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	// too early
	return reconcile.Result{
		Requeue:      true,
		RequeueAfter: r.getDurationUntilNextRetryForSucceeded(inst),
	}, nil
}

func (r *retryHelper) addReconcileAnnotation(ctx context.Context, inst *lsv1alpha1.Installation) error {
	logger, ctx := logging.FromContextOrNew(ctx, nil)

	lsv1alpha1helper.SetOperation(&inst.ObjectMeta, lsv1alpha1.ReconcileOperation)
	metav1.SetMetaDataAnnotation(&inst.ObjectMeta, lsv1alpha1.ReconcileReasonAnnotation, reconcileReasonRetry)

	if err := r.writer.UpdateInstallation(ctx, read_write_layer.W000029, inst); err != nil {
		logger.Error(err, "failed to trigger retry of installation")
		return err
	}

	return nil
}

func (r *retryHelper) maxNumberOfRetriesDoneForFailed(inst *lsv1alpha1.Installation) bool {
	alreadyExecutedRetries := 0
	if inst.Status.AutomaticReconcileStatus != nil {
		alreadyExecutedRetries = inst.Status.AutomaticReconcileStatus.NumberOfReconciles
	}
	return inst.Spec.AutomaticReconcile.FailedReconcile.NumberOfReconciles != nil &&
		*inst.Spec.AutomaticReconcile.FailedReconcile.NumberOfReconciles <= alreadyExecutedRetries
}

func (r *retryHelper) isNextRetryDueForFailed(inst *lsv1alpha1.Installation) bool {
	return r.now().After(r.getNextRetryTimeForFailed(inst))
}

func (r *retryHelper) isNextRetryDueForNewAndSucceeded(inst *lsv1alpha1.Installation) bool {
	return r.now().After(r.getNextRetryTimeForSucceeded(inst))
}

func (r *retryHelper) getDurationUntilNextRetryForFailed(inst *lsv1alpha1.Installation) time.Duration {
	return r.getNextRetryTimeForFailed(inst).Sub(r.now())
}

func (r *retryHelper) getDurationUntilNextRetryForSucceeded(inst *lsv1alpha1.Installation) time.Duration {
	return r.getNextRetryTimeForSucceeded(inst).Sub(r.now())
}

func (r *retryHelper) getNextRetryTimeForFailed(inst *lsv1alpha1.Installation) time.Time {
	lastRetryTime := inst.Status.AutomaticReconcileStatus.LastReconcileTime.Time // TODO
	return lastRetryTime.Add(r.getRetryIntervalForFailed(inst))
}

func (r *retryHelper) getNextRetryTimeForSucceeded(inst *lsv1alpha1.Installation) time.Time {
	lastRetryTime := inst.Status.AutomaticReconcileStatus.LastReconcileTime.Time // TODO
	return lastRetryTime.Add(r.getRetryIntervalForSucceeded(inst))
}

func (r *retryHelper) getRetryIntervalForFailed(inst *lsv1alpha1.Installation) time.Duration {
	retryInterval := inst.Spec.AutomaticReconcile.FailedReconcile.Interval
	if retryInterval == nil {
		return defaultRetryDurationForFailed
	}
	return retryInterval.Duration
}

func (r *retryHelper) getRetryIntervalForSucceeded(inst *lsv1alpha1.Installation) time.Duration {
	retryInterval := inst.Spec.AutomaticReconcile.SucceededReconcile.Interval
	if retryInterval == nil {
		return defaultRetryDurationForNewAndSucceeded
	}
	return retryInterval.Duration
}

func (r *retryHelper) now() time.Time {
	return r.clock.Now()
}

func (r *retryHelper) metaNow() metav1.Time {
	return metav1.Time{Time: r.now()}
}
