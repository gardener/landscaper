// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"fmt"
	"reflect"
	"time"

	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"

	"github.com/gardener/landscaper/pkg/landscaper/installations/executions"

	"github.com/google/uuid"

	"github.com/gardener/component-cli/ociclient/cache"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
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
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	operation "github.com/gardener/landscaper/pkg/landscaper/operation"
)

const (
	cacheIdentifier = "landscaper-installation-Controller"
)

// NewController creates a new Controller that reconciles Installation resources.
func NewController(logger logging.Logger,
	kubeClient client.Client,
	scheme *runtime.Scheme,
	eventRecorder record.EventRecorder,
	overwriter componentoverwrites.Overwriter,
	lsConfig *config.LandscaperConfiguration) (reconcile.Reconciler, error) {

	ctrl := &Controller{
		log:                 logger,
		LsConfig:            lsConfig,
		ComponentOverwriter: overwriter,
	}

	if lsConfig != nil && lsConfig.Registry.OCI != nil {
		var err error
		ctrl.SharedCache, err = cache.NewCache(logger.Logr(), utils.ToOCICacheOptions(lsConfig.Registry.OCI.Cache, cacheIdentifier)...)
		if err != nil {
			return nil, err
		}
		logger.Debug("setup shared components registry  cache")
	}

	op := operation.NewOperation(kubeClient, scheme, eventRecorder)
	ctrl.Operation = *op
	return ctrl, nil
}

// NewTestActuator creates a new Controller that is only meant for testing.
func NewTestActuator(op operation.Operation, logger logging.Logger, configuration *config.LandscaperConfiguration) *Controller {
	a := &Controller{
		log:       logger,
		Operation: op,
		LsConfig:  configuration,
	}
	return a
}

// Controller is the controller that reconciles a installation resource.
type Controller struct {
	operation.Operation
	log                 logging.Logger
	LsConfig            *config.LandscaperConfiguration
	SharedCache         cache.Cache
	ComponentOverwriter componentoverwrites.Overwriter
}

func (c *Controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	if !utils.IsNewReconcile() {
		return c.reconcileOld(ctx, req)
	} else {
		return c.reconcileNew(ctx, req)
	}
}

func (c *Controller) reconcileNew(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger, ctx := c.log.StartReconcileAndAddToContext(ctx, req)

	inst := &lsv1alpha1.Installation{}
	if err := read_write_layer.GetInstallation(ctx, c.Client(), req.NamespacedName, inst); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// default the installation as it not done by the Controller runtime
	api.LandscaperScheme.Default(inst)

	if inst.DeletionTimestamp.IsZero() && !kutil.HasFinalizer(inst, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(inst, lsv1alpha1.LandscaperFinalizer)
		if err := c.Writer().UpdateInstallation(ctx, read_write_layer.W000107, inst); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.InterruptOperation) {
		if err := c.handleInterruptOperation(ctx, inst); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if !installations.IsRootInstallation(inst) && lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ReconcileOperation) {
		// only root installations could be triggered with operation annotation to prevent that end users interfere with overall
		// algorithm
		logger.Info("Removing reconcile annotation from non-root installation. A reconcile annotation at a non-root installation has no effect")
		if err := c.removeReconcileAnnotation(ctx, inst); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil

	}

	// generate new jobID
	isFirstDelete := !inst.DeletionTimestamp.IsZero() && !lsv1alpha1helper.IsDeletionInstallationPhase(inst.Status.InstallationPhase)
	if installations.IsRootInstallation(inst) &&
		(lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ReconcileOperation) || isFirstDelete) &&
		inst.Status.JobID == inst.Status.JobIDFinished {

		inst.Status.JobID = uuid.New().String()
		if err := c.Writer().UpdateInstallationStatus(ctx, read_write_layer.W000082, inst); err != nil {
			return reconcile.Result{}, err
		}

		if err := c.removeReconcileAnnotation(ctx, inst); err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	// handle reconcile
	if inst.Status.JobID != inst.Status.JobIDFinished {

		err := c.handleReconcilePhase(ctx, inst)
		return reconcile.Result{}, err

	} else {
		// job finished; nothing to do
		return reconcile.Result{}, nil
	}
}

func (c *Controller) reconcileOld(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger, ctx := c.log.StartReconcileAndAddToContext(ctx, req)

	inst := &lsv1alpha1.Installation{}
	if err := read_write_layer.GetInstallation(ctx, c.Client(), req.NamespacedName, inst); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// don't reconcile if ignore annotation is set and installation is not currently running
	if lsv1alpha1helper.HasIgnoreAnnotation(inst.ObjectMeta) && lsv1alpha1helper.IsCompletedInstallationPhase(inst.Status.Phase) {
		logger.Debug("skipping reconcile due to ignore annotation")
		return reconcile.Result{}, nil
	}

	// default the installation as it not done by the Controller runtime
	api.LandscaperScheme.Default(inst)

	oldInst := inst.DeepCopy()

	if inst.DeletionTimestamp.IsZero() && !kutil.HasFinalizer(inst, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(inst, lsv1alpha1.LandscaperFinalizer)
		if err := c.Writer().UpdateInstallation(ctx, read_write_layer.W000010, inst); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if !inst.DeletionTimestamp.IsZero() {
		err := c.handleDelete(ctx, inst)
		return reconcile.Result{}, c.handleError(ctx, err, oldInst, inst, true)
	}

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
		err := c.forceReconcile(ctx, inst)
		return reconcile.Result{}, c.handleError(ctx, err, oldInst, inst, false)
	}

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.AbortOperation) {
		// todo: handle abort..
		logger.Info("do abort")
	}

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ReconcileOperation) {
		err := c.reconcile(ctx, inst)
		return reconcile.Result{}, c.handleError(ctx, err, oldInst, inst, false)
	} else if !lsv1alpha1helper.IsCompletedInstallationPhase(inst.Status.Phase) || inst.Status.ObservedGeneration != inst.Generation {
		err := c.reconcile(ctx, inst)
		return reconcile.Result{}, c.handleError(ctx, err, oldInst, inst, false)
	} else {
		// check whether the current phase does still match the combined phase of subinstallations and executions
		if err := c.handleSubComponentPhaseChanges(ctx, inst); err != nil {
			return reconcile.Result{}, c.handleError(ctx, err, oldInst, inst, false)
		}
		return reconcile.Result{}, nil
	}
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

	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyReconciledResource, client.ObjectKeyFromObject(inst).String()})

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
		logger.Debug("execution phase mismatch", "phase", string(inst.Status.Phase), "combinedPhase", string(cp))

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

func (c *Controller) handleError(ctx context.Context, err lserrors.LsError, oldInst, inst *lsv1alpha1.Installation, isDelete bool) error {
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyReconciledResource, client.ObjectKeyFromObject(inst).String()})

	inst.Status.LastError = lserrors.TryUpdateLsError(inst.Status.LastError, err)
	// if successfully deleted we could not update the object
	if isDelete && err == nil {
		inst2 := &lsv1alpha1.Installation{}
		if err2 := read_write_layer.GetInstallation(ctx, c.Client(), kutil.ObjectKey(inst.Name, inst.Namespace), inst2); err2 != nil {
			if apierrors.IsNotFound(err2) {
				return nil
			}
		}
	}

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
				logger.Info(fmt.Sprintf("unable to update status: %s", err2.Error()))
			} else {
				logger.Error(err2, "unable to update status")
			}
			if err == nil {
				return err2
			}
		}
	}

	return err
}

func (c *Controller) compareJobIDs(predecessorMap, predecessorMapNew map[string]*installations.InstallationBase) bool {
	if len(predecessorMap) != len(predecessorMapNew) {
		return false
	}

	for name, oldPredecessor := range predecessorMap {
		newPredecessor := predecessorMapNew[name]
		if newPredecessor == nil {
			return false
		}

		if oldPredecessor.Info.Status.JobID != newPredecessor.Info.Status.JobID {
			return false
		}
	}

	return true
}

func (c *Controller) handleInterruptOperation(ctx context.Context, inst *lsv1alpha1.Installation) error {
	delete(inst.Annotations, lsv1alpha1.OperationAnnotation)
	if err := c.Writer().UpdateInstallation(ctx, read_write_layer.W000097, inst); err != nil {
		return err
	}

	exec, err := executions.GetExecutionForInstallation(ctx, c.Client(), inst)
	if err != nil {
		return err
	}

	if exec != nil {
		lsv1alpha1helper.SetOperation(&exec.ObjectMeta, lsv1alpha1.InterruptOperation)
		lsv1alpha1helper.Touch(&exec.ObjectMeta)

		if err = c.Writer().UpdateExecution(ctx, read_write_layer.W000098, exec); err != nil {
			return err
		}
	}

	subInsts, err := installations.ListSubinstallations(ctx, c.Client(), inst)
	if err != nil {
		return nil
	}

	for _, subInst := range subInsts {
		lsv1alpha1helper.SetOperation(&subInst.ObjectMeta, lsv1alpha1.InterruptOperation)
		lsv1alpha1helper.Touch(&subInst.ObjectMeta)

		if err = c.Writer().UpdateInstallation(ctx, read_write_layer.W000099, subInst); err != nil {
			return err
		}
	}

	return nil
}

func (c *Controller) setInstallationPhaseAndUpdate(ctx context.Context, inst *lsv1alpha1.Installation,
	phase lsv1alpha1.InstallationPhase, lsError lserrors.LsError, writeID read_write_layer.WriteID) lserrors.LsError {

	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyReconciledResource, client.ObjectKeyFromObject(inst).String()})

	if lsError != nil {
		logger.Error(lsError, "setInstallationPhaseAndUpdate:"+lsError.Error())
	}

	inst.Status.LastError = lserrors.TryUpdateLsError(inst.Status.LastError, lsError)

	if inst.Status.LastError != nil {
		lastErr := inst.Status.LastError
		c.EventRecorder().Event(inst, corev1.EventTypeWarning, lastErr.Reason, lastErr.Message)
	}

	inst.Status.InstallationPhase = phase
	if phase == lsv1alpha1.InstallationPhaseFailed ||
		phase == lsv1alpha1.InstallationPhaseSucceeded ||
		phase == lsv1alpha1.InstallationPhaseDeleteFailed {
		inst.Status.JobIDFinished = inst.Status.JobID
	}

	if err := c.Writer().UpdateInstallationStatus(ctx, writeID, inst); err != nil {
		if inst.Status.InstallationPhase == lsv1alpha1.InstallationPhaseDeleting {
			// recheck if already deleted
			instRecheck := &lsv1alpha1.Installation{}
			errRecheck := read_write_layer.GetInstallation(ctx, c.Client(), kutil.ObjectKey(inst.Name, inst.Namespace), instRecheck)
			if errRecheck != nil && apierrors.IsNotFound(errRecheck) {
				return nil
			}
		}

		logger.Error(err, "unable to update installation status")
		if lsError == nil {
			return lserrors.NewWrappedError(err, "setInstallationPhaseAndUpdate", "UpdateInstallationStatus", err.Error())
		}
	}

	return lsError
}

func (c *Controller) checkForDuplicateExports(ctx context.Context, inst *lsv1alpha1.Installation) error {
	// fetch all installations in the same namespace with the same parent
	var selector client.ListOption
	if parent, ok := inst.Labels[lsv1alpha1.EncompassedByLabel]; ok {
		selector = client.MatchingLabels(map[string]string{
			lsv1alpha1.EncompassedByLabel: parent,
		})
	} else {
		r, err := labels.NewRequirement(lsv1alpha1.EncompassedByLabel, selection.DoesNotExist, nil)
		if err != nil {
			return err
		}
		selector = client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(*r)}
	}
	siblingList := &lsv1alpha1.InstallationList{}
	err := read_write_layer.ListInstallations(ctx, c.Client(), siblingList, client.InNamespace(inst.Namespace), selector)
	if err != nil {
		return err
	}

	return utils.CheckForDuplicateExports(inst, siblingList.Items)
}
