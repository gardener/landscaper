// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lserrors "github.com/gardener/landscaper/apis/errors"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/execution"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
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
	logger := c.log.WithValues("resource", req.NamespacedName)
	logger.V(5).Info("reconcile")

	exec := &lsv1alpha1.Execution{}
	if err := c.client.Get(ctx, req.NamespacedName, exec); err != nil {
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

	errHdl := HandleErrorFunc(logger, c.client, c.eventRecorder, exec)

	if err := HandleAnnotationsAndGeneration(ctx, logger, c.client, exec); err != nil {
		return reconcile.Result{}, errHdl(ctx, err)
	}

	if lsv1alpha1helper.IsCompletedExecutionPhase(exec.Status.Phase) {
		op := execution.NewOperation(operation.NewOperation(logger, c.client, c.scheme, c.eventRecorder), exec, false)
		err := op.HandleDeployItemPhaseAndGenerationChanges(ctx, logger)
		if err != nil {
			return reconcile.Result{}, lserrors.NewWrappedError(err, "Reconcile", "HandleDeployItemPhaseAndGenerationChanges", err.Error())
		}
		if lsv1alpha1helper.IsCompletedExecutionPhase(exec.Status.Phase) {
			return reconcile.Result{}, nil
		}
	}

	return reconcile.Result{}, errHdl(ctx, c.Ensure(ctx, logger, exec))
}

func (c *controller) Ensure(ctx context.Context, log logr.Logger, exec *lsv1alpha1.Execution) error {
	forceReconcile := lsv1alpha1helper.HasOperation(exec.ObjectMeta, lsv1alpha1.ForceReconcileOperation)
	op := execution.NewOperation(operation.NewOperation(log, c.client, c.scheme, c.eventRecorder), exec,
		forceReconcile)

	if exec.DeletionTimestamp.IsZero() && !kubernetes.HasFinalizer(exec, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(exec, lsv1alpha1.LandscaperFinalizer)
		if err := c.client.Update(ctx, exec); err != nil {
			return lserrors.NewError("Reconcile", "AddFinalizer", err.Error())
		}
	}

	if !exec.DeletionTimestamp.IsZero() {
		return op.Delete(ctx)
	}

	return op.Reconcile(ctx)
}

// HandleAnnotationsAndGeneration is meant to be called at the beginning of the reconcile loop.
// If a reconcile is needed due to the reconcile annotation or a change in the generation, it will set the phase to Init and remove the reconcile annotation.
// Returns: an error, if updating the execution failed, nil otherwise
func HandleAnnotationsAndGeneration(ctx context.Context, log logr.Logger, c client.Client, exec *lsv1alpha1.Execution) error {
	hasReconcileAnnotation := lsv1alpha1helper.HasOperation(exec.ObjectMeta, lsv1alpha1.ReconcileOperation)
	if hasReconcileAnnotation || exec.Status.ObservedGeneration != exec.Generation {
		// reconcile necessary due to one of
		// - reconcile annotation
		// - outdated generation
		log.V(5).Info("reconcile required, setting observed generation and phase", "reconcileAnnotation", hasReconcileAnnotation, "observedGeneration", exec.Status.ObservedGeneration, "generation", exec.Generation)
		exec.Status.ObservedGeneration = exec.Generation
		exec.Status.Phase = lsv1alpha1.ExecutionPhaseInit

		log.V(7).Info("updating status")
		if err := c.Status().Update(ctx, exec); err != nil {
			return err
		}
		log.V(7).Info("successfully updated status")
	}
	if hasReconcileAnnotation {
		log.V(5).Info("removing reconcile annotation")
		delete(exec.ObjectMeta.Annotations, lsv1alpha1.OperationAnnotation)
		log.V(7).Info("updating metadata")
		if err := c.Update(ctx, exec); err != nil {
			return err
		}
		log.V(7).Info("successfully updated metadata")
	}

	// also reset the phase when the force reconcile annotation is present.
	// Otherwise we would never bbe able to leave a final phase
	if lsv1alpha1helper.HasOperation(exec.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
		if lsv1alpha1helper.IsCompletedExecutionPhase(exec.Status.Phase) {
			exec.Status.Phase = lsv1alpha1.ExecutionPhaseInit
			log.V(7).Info("updating status")
			if err := c.Status().Update(ctx, exec); err != nil {
				return err
			}
			log.V(7).Info("successfully updated status")
		}
	}
	return nil
}

// HandleErrorFunc returns a error handler func for deployers.
// The functions automatically sets the phase for long running errors and updates the status accordingly.
func HandleErrorFunc(log logr.Logger, client client.Client, eventRecorder record.EventRecorder, exec *lsv1alpha1.Execution) func(ctx context.Context, err error) error {
	old := exec.DeepCopy()
	return func(ctx context.Context, err error) error {
		if err == nil && reflect.DeepEqual(old.Status.LastError, exec.Status.LastError) {
			// don't set LastError to nil if it already has been overwritten and no error occured
			// this is needed to allow the Reconcile function to set LastError without returning an Error
			// (meaning the execution is failed but the reconciliation itself was successful)
			// if an error is returned, any changes to LastError will be overwritten
			exec.Status.LastError = lserrors.TryUpdateError(old.Status.LastError, err)
		}
		exec.Status.Phase = lsv1alpha1.ExecutionPhase(lserrors.GetPhaseForLastError(
			lsv1alpha1.ComponentInstallationPhase(exec.Status.Phase),
			exec.Status.LastError,
			5*time.Minute))
		if exec.Status.LastError != nil {
			lastErr := exec.Status.LastError
			eventRecorder.Event(exec, corev1.EventTypeWarning, lastErr.Reason, lastErr.Message)
		}

		if !reflect.DeepEqual(old.Status, exec.Status) {
			if err2 := client.Status().Update(ctx, exec); err2 != nil {
				if apierrors.IsConflict(err2) { // reduce logging
					log.V(5).Info(fmt.Sprintf("unable to update status: %s", err2.Error()))
				} else {
					log.Error(err2, "unable to update status")
				}
				// retry on conflict
				if err != nil {
					return err2
				}
			}
		}
		return err
	}
}
