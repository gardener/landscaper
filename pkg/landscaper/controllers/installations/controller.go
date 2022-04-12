// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lserrors "github.com/gardener/landscaper/apis/errors"

	"github.com/gardener/landscaper/pkg/utils"

	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/installations/exports"
	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
)

const (
	cacheIdentifier = "landscaper-installation-Controller"
)

// NewController creates a new Controller that reconciles Installation resources.
func NewController(log logr.Logger,
	kubeClient client.Client,
	scheme *runtime.Scheme,
	eventRecorder record.EventRecorder,
	overwriter componentoverwrites.Overwriter,
	lsConfig *config.LandscaperConfiguration) (reconcile.Reconciler, error) {

	ctrl := &Controller{
		LsConfig:            lsConfig,
		ComponentOverwriter: overwriter,
	}

	if lsConfig != nil && lsConfig.Registry.OCI != nil {
		var err error
		ctrl.SharedCache, err = cache.NewCache(log, utils.ToOCICacheOptions(lsConfig.Registry.OCI.Cache, cacheIdentifier)...)
		if err != nil {
			return nil, err
		}
		log.V(3).Info("setup shared components registry  cache")
	}

	op := operation.NewOperation(log, kubeClient, scheme, eventRecorder)
	ctrl.Operation = *op
	return ctrl, nil
}

// NewTestActuator creates a new Controller that is only meant for testing.
func NewTestActuator(op operation.Operation, configuration *config.LandscaperConfiguration) *Controller {
	a := &Controller{
		Operation: op,
		LsConfig:  configuration,
	}
	return a
}

// Controller is the controller that reconciles a installation resource.
type Controller struct {
	operation.Operation
	LsConfig            *config.LandscaperConfiguration
	SharedCache         cache.Cache
	ComponentOverwriter componentoverwrites.Overwriter
}

func (c *Controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := c.Log().WithValues("installation", req.NamespacedName.String())
	ctx = logr.NewContext(ctx, logger)
	logger.V(5).Info("reconcile", "resource", req.NamespacedName)

	inst := &lsv1alpha1.Installation{}
	if err := read_write_layer.GetInstallation(ctx, c.Client(), req.NamespacedName, inst); err != nil {
		if apierrors.IsNotFound(err) {
			c.Log().V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// don't reconcile if ignore annotation is set and installation is not currently running
	if lsv1alpha1helper.HasIgnoreAnnotation(inst.ObjectMeta) && lsv1alpha1helper.IsCompletedInstallationPhase(inst.Status.Phase) {
		logger.V(7).Info("skipping reconcile due to ignore annotation")
		return reconcile.Result{}, nil
	}

	// default the installation as it not done by the Controller runtime
	api.LandscaperScheme.Default(inst)

	oldInst := inst.DeepCopy()

	if inst.DeletionTimestamp.IsZero() && !kubernetes.HasFinalizer(inst, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(inst, lsv1alpha1.LandscaperFinalizer)
		if err := c.Writer().UpdateInstallation(ctx, read_write_layer.W000010, inst); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if !inst.DeletionTimestamp.IsZero() {
		err := c.handleDelete(ctx, inst)
		return reconcile.Result{}, c.handleError(ctx, err, oldInst, inst)
	}

	// remove the reconcile annotation if it exists
	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ReconcileOperation) {
		delete(inst.Annotations, lsv1alpha1.OperationAnnotation)
		if err := c.Writer().UpdateInstallation(ctx, read_write_layer.W000009, inst); err != nil {
			return reconcile.Result{}, err
		}

		err := c.reconcile(ctx, inst)
		return reconcile.Result{}, c.handleError(ctx, err, oldInst, inst)
	}

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
		err := c.forceReconcile(ctx, inst)
		return reconcile.Result{}, c.handleError(ctx, err, oldInst, inst)
	}

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.AbortOperation) {
		// todo: handle abort..
		c.Log().Info("do abort")
	}

	if lsv1alpha1helper.IsCompletedInstallationPhase(inst.Status.Phase) && inst.Status.ObservedGeneration == inst.Generation {
		// check whether the current phase does still match the combined phase of subinstallations and executions
		if err := c.handleSubComponentPhaseChanges(ctx, inst); err != nil {
			return reconcile.Result{}, c.handleError(ctx, err, oldInst, inst)
		}
		return reconcile.Result{}, nil
	}

	err := c.reconcile(ctx, inst)
	return reconcile.Result{}, c.handleError(ctx, err, oldInst, inst)
}

// initPrerequisites prepares installation operations by fetching context and registries, resolving the blueprint and creating an internal installation.
// It does not modify the installation resource in the cluster in any way.
func (c *Controller) initPrerequisites(ctx context.Context, inst *lsv1alpha1.Installation) (*installations.Operation, lserrors.LsError) {
	currOp := "InitPrerequisites"
	op := c.Operation.Copy()

	lsCtx, err := installations.GetInstallationContext(ctx, c.Client(), inst, c.ComponentOverwriter)
	if err != nil {
		return nil, lserrors.NewWrappedError(err, currOp, "CalculateContext", err.Error())
	}

	if err := c.SetupRegistries(ctx, op, append(lsCtx.External.RegistryPullSecrets(), inst.Spec.RegistryPullSecrets...), inst); err != nil {
		return nil, lserrors.NewWrappedError(err, currOp, "SetupRegistries", err.Error())
	}

	intBlueprint, err := blueprints.Resolve(ctx, op.ComponentsRegistry(), lsCtx.External.ComponentDescriptorRef(), inst.Spec.Blueprint)
	if err != nil {
		return nil, lserrors.NewWrappedError(err, currOp, "ResolveBlueprint", err.Error())
	}

	internalInstallation, err := installations.New(inst, intBlueprint)
	if err != nil {
		err = fmt.Errorf("unable to create internal representation of installation: %w", err)
		return nil, lserrors.NewWrappedError(err,
			currOp, "InitInstallation", err.Error())
	}

	instOp, err := installations.NewOperationBuilder(internalInstallation).
		WithOperation(op).
		WithContext(lsCtx).
		Build(ctx)
	if err != nil {
		err = fmt.Errorf("unable to create installation operation: %w", err)
		return nil, lserrors.NewWrappedError(err,
			currOp, "InitInstallationOperation", err.Error())
	}
	instOp.SetOverwriter(c.ComponentOverwriter)
	return instOp, nil
}

// HandleSubComponentPhaseChanges updates the phase of the given installation, if its phase doesn't match the combined phase of its subinstallations/executions anymore
func (c *Controller) handleSubComponentPhaseChanges(
	ctx context.Context,
	inst *lsv1alpha1.Installation) lserrors.LsError {
	logger := logr.FromContext(ctx)

	execRef := inst.Status.ExecutionReference
	phases := []lsv1alpha1.ComponentInstallationPhase{}
	if execRef != nil {
		exec := &lsv1alpha1.Execution{}
		err := read_write_layer.GetExecution(ctx, c.Client(), execRef.NamespacedName(), exec)
		if err != nil {
			message := fmt.Sprintf("error getting execution for installation %s/%s", inst.Namespace, inst.Name)
			return lserrors.NewWrappedError(err, "handleSubComponentPhaseChanges", "GetExecution", message)
		}
		phases = append(phases, lsv1alpha1.ComponentInstallationPhase(exec.Status.Phase))
	}
	subinsts, err := installations.ListSubinstallations(ctx, c.Client(), inst)
	if err != nil {
		message := fmt.Sprintf("error fetching subinstallations for installation %s/%s", inst.Namespace, inst.Name)
		return lserrors.NewWrappedError(err, "handleSubComponentPhaseChanges", "ListSubinstallations", message)
	}
	for _, sub := range subinsts {
		phases = append(phases, sub.Status.Phase)
	}
	if len(phases) == 0 {
		// Installation contains neither an execution nor subinstallations, so the phase can't be out of sync.
		return nil
	}
	cp := lsv1alpha1helper.CombinedInstallationPhase(phases...)
	if inst.Status.Phase != cp {
		// Phase is completed but doesn't fit to the deploy items' phases
		logger.V(5).Info("execution phase mismatch", "phase", string(inst.Status.Phase), "combinedPhase", string(cp))

		// get operation
		var err error
		instOp, err := c.initPrerequisites(ctx, inst)
		if err != nil {
			message := fmt.Sprintf("unable to construct operation for installation %s/%s", inst.Namespace, inst.Name)
			return lserrors.NewWrappedError(err, "handleSubComponentPhaseChanges", "initPrerequisites", message)
		}

		if cp == lsv1alpha1.ComponentPhaseSucceeded {
			// recompute exports
			dataExports, targetExports, err := exports.NewConstructor(instOp).Construct(ctx)
			if err != nil {
				return instOp.NewError(err, "ConstructExports", err.Error())
			}
			if err := instOp.CreateOrUpdateExports(ctx, dataExports, targetExports); err != nil {
				return instOp.NewError(err, "CreateOrUpdateExports", err.Error())
			}
		}

		// update status
		err = instOp.UpdateInstallationStatus(ctx, inst, cp)
		if err != nil {
			message := fmt.Sprintf("error updating installation status for installation %s/%s", inst.Namespace, inst.Name)
			return lserrors.NewWrappedError(err, "handleSubComponentPhaseChanges", "UpdateInstallationStatus", message)
		}

		if cp == lsv1alpha1.ComponentPhaseSucceeded {
			// trigger dependent installations
			err = instOp.TriggerDependents(ctx)
			if err != nil {
				return instOp.NewError(err, "TriggerDependants", err.Error())
			}
		}

		return nil
	}
	return nil
}

func (c *Controller) handleError(ctx context.Context, err lserrors.LsError, oldInst, inst *lsv1alpha1.Installation) error {
	inst.Status.LastError = lserrors.TryUpdateLsError(inst.Status.LastError, err)

	inst.Status.Phase = lserrors.GetPhaseForLastError(
		inst.Status.Phase,
		inst.Status.LastError,
		5*time.Minute)

	if inst.Status.LastError != nil {
		lastErr := inst.Status.LastError
		c.EventRecorder().Event(inst, corev1.EventTypeWarning, lastErr.Reason, lastErr.Message)
	}

	if !reflect.DeepEqual(oldInst.Status, inst.Status) {
		if err2 := c.Writer().UpdateInstallationStatus(ctx, read_write_layer.W000015, inst); err2 != nil {
			if apierrors.IsConflict(err2) { // reduce logging
				c.Log().V(5).Info(fmt.Sprintf("unable to update status: %s", err2.Error()))
			} else {
				c.Log().Error(err2, "unable to update status")
			}
			// retry on conflict
			if err == nil {
				return err2
			}
		}
	}
	return err
}
