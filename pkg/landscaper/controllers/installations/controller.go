// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"fmt"
	"reflect"

	"github.com/gardener/landscaper/pkg/components/registries"

	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lserrors "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/api"
	cnudieutils "github.com/gardener/landscaper/pkg/components/cnudie/utils"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/lock"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

const (
	cacheIdentifier = "landscaper-installation-Controller"
)

// NewController creates a new Controller that reconciles Installation resources.
func NewController(hostClient client.Client, logger logging.Logger,
	kubeClient client.Client,
	scheme *runtime.Scheme,
	eventRecorder record.EventRecorder,
	lsConfig *config.LandscaperConfiguration,
	maxNumberOfWorkers int,
	lockingEnabled bool,
	callerName string) (reconcile.Reconciler, error) {

	ws := utils.NewWorkerCounter(maxNumberOfWorkers)

	ctrl := &Controller{
		hostClient:     hostClient,
		log:            logger,
		clock:          clock.RealClock{},
		LsConfig:       lsConfig,
		workerCounter:  ws,
		lockingEnabled: lockingEnabled,
		callerName:     callerName,
	}

	if lsConfig != nil && lsConfig.Registry.OCI != nil {
		var err error
		ctrl.SharedCache, err = cache.NewCache(logger.Logr(), cnudieutils.ToOCICacheOptions(lsConfig.Registry.OCI.Cache, cacheIdentifier)...)
		if err != nil {
			return nil, err
		}
		logger.Debug("setup shared components registry  cache")
	}

	registries.SetOCMLibraryMode(lsConfig.UseOCMLib)

	op := operation.NewOperation(kubeClient, scheme, eventRecorder)
	ctrl.Operation = *op
	return ctrl, nil
}

// NewTestActuator creates a new Controller that is only meant for testing.
func NewTestActuator(op operation.Operation, hostClient client.Client, logger logging.Logger, passiveClock clock.PassiveClock,
	configuration *config.LandscaperConfiguration, callerName string) *Controller {

	return &Controller{
		log:            logger,
		clock:          passiveClock,
		Operation:      op,
		LsConfig:       configuration,
		workerCounter:  utils.NewWorkerCounter(1000),
		hostClient:     hostClient,
		lockingEnabled: lock.IsLockingEnabledForMainControllers(configuration),
		callerName:     callerName,
	}
}

// Controller is the controller that reconciles a installation resource.
type Controller struct {
	operation.Operation
	hostClient     client.Client
	log            logging.Logger
	clock          clock.PassiveClock
	LsConfig       *config.LandscaperConfiguration
	SharedCache    cache.Cache
	workerCounter  *utils.WorkerCounter
	lockingEnabled bool
	callerName     string
}

func (c *Controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger, ctx := c.log.StartReconcileAndAddToContext(ctx, req)

	c.workerCounter.EnterWithLog(logger, 70, "installations")
	defer c.workerCounter.Exit()

	metadata := utils.EmptyInstallationMetadata()
	if err := c.Client().Get(ctx, req.NamespacedName, metadata); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Debug(err.Error())
			return reconcile.Result{}, nil
		}
		return utils.LogHelper{}.LogStandardErrorAndGetReconcileResult(ctx, err)
	}

	if c.lockingEnabled {
		locker := lock.NewLocker(c.Client(), c.hostClient, c.callerName)
		syncObject, err := locker.LockInstallation(ctx, metadata)
		if err != nil {
			return utils.LogHelper{}.LogStandardErrorAndGetReconcileResult(ctx, err)
		}

		if syncObject == nil {
			return locker.NotLockedResult()
		}

		defer func() {
			locker.Unlock(ctx, syncObject)
		}()
	}

	inst := &lsv1alpha1.Installation{}
	if err := read_write_layer.GetInstallation(ctx, c.Client(), req.NamespacedName, inst); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info(err.Error())
			return reconcile.Result{}, nil
		}
		return utils.LogHelper{}.LogStandardErrorAndGetReconcileResult(ctx, err)
	}

	// default the installation as it not done by the Controller runtime
	if err := c.updateInstallationWithDefaults(ctx, inst); err != nil {
		return utils.LogHelper{}.LogStandardErrorAndGetReconcileResult(ctx, err)
	}

	return c.handleAutomaticReconcile(ctx, inst)
}

func (c *Controller) updateInstallationWithDefaults(ctx context.Context, inst *lsv1alpha1.Installation) error {
	logger, ctx := logging.FromContextOrNew(ctx, nil)
	oldInst := inst.DeepCopy()

	// default the installation as it not done by the Controller runtime
	api.LandscaperScheme.Default(inst)

	if !reflect.DeepEqual(&inst.Spec, &oldInst.Spec) {
		if err := c.Writer().UpdateInstallation(ctx, read_write_layer.W000065, inst); err != nil {
			logger.Error(err, "failed to update installation with defaults")
			return err
		}
	}

	return nil
}

func (c *Controller) handleAutomaticReconcile(ctx context.Context, inst *lsv1alpha1.Installation) (reconcile.Result, error) {
	logger, _ := logging.FromContextOrNew(ctx, nil, lc.KeyMethod, "handleAutomaticReconcile")

	retryHelper := newRetryHelper(c.Client(), c.clock)

	if err := retryHelper.preProcessRetry(ctx, inst); err != nil {
		return utils.LogHelper{}.LogStandardErrorAndGetReconcileResult(ctx, err)
	}

	result, err := c.reconcileInstallation(ctx, inst)

	result, err = retryHelper.recomputeRetry(ctx, inst, result, err)
	if err != nil {
		logger.Error(err, "recomputeRetry failed")
	}

	return result, err
}

func (c *Controller) reconcileInstallation(ctx context.Context, inst *lsv1alpha1.Installation) (reconcile.Result, error) {
	logger, ctx := logging.FromContextOrNew(ctx, nil)

	if inst.DeletionTimestamp.IsZero() && !kutil.HasFinalizer(inst, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(inst, lsv1alpha1.LandscaperFinalizer)
		if err := c.Writer().UpdateInstallation(ctx, read_write_layer.W000107, inst); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if err := installations.NewInstallationTrigger(c.Client(), inst).TriggerDependents(ctx); err != nil {
		return reconcile.Result{}, err
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
	isFirstDelete := !inst.DeletionTimestamp.IsZero() && !inst.Status.InstallationPhase.IsDeletion()
	if installations.IsRootInstallation(inst) &&
		(lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ReconcileOperation) || isFirstDelete) &&
		inst.Status.JobID == inst.Status.JobIDFinished {

		inst.Status.JobID = uuid.New().String()
		inst.Status.TransitionTimes = utils.NewTransitionTimes()

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
		return utils.LogHelper{}.LogErrorAndGetReconcileResult(ctx, err)
	} else {
		// job finished; nothing to do
		return reconcile.Result{}, nil
	}
}

// initPrerequisites prepares installation operations by fetching context and registries, resolving the blueprint and creating an internal installation.
// It does not modify the installation resource in the cluster in any way.
func (c *Controller) initPrerequisites(ctx context.Context, inst *lsv1alpha1.Installation) (*installations.Operation, lserrors.LsError) {
	currOp := "InitPrerequisites"
	op := c.Operation.Copy()

	lsCtx, err := installations.GetInstallationContext(ctx, c.Client(), inst)
	if err != nil {
		return nil, lserrors.NewWrappedError(err, currOp, "CalculateContext", err.Error())
	}

	if err := c.SetupRegistries(ctx, op, lsCtx.External.Context, lsCtx.External.RegistryPullSecrets(), inst); err != nil {
		return nil, lserrors.NewWrappedError(err, currOp, "SetupRegistries", err.Error())
	}

	intBlueprint, err := blueprints.Resolve(ctx, op.ComponentsRegistry(), lsCtx.External.ComponentDescriptorRef(), inst.Spec.Blueprint)
	if err != nil {
		return nil, lserrors.NewWrappedError(err, currOp, "ResolveBlueprint", err.Error())
	}

	internalInstallation := installations.NewInstallationImportsAndBlueprint(inst, intBlueprint)

	instOp, err := installations.NewOperationBuilder(internalInstallation).
		WithOperation(op).
		WithContext(lsCtx).
		Build(ctx)
	if err != nil {
		err = fmt.Errorf("unable to create installation operation: %w", err)
		return nil, lserrors.NewWrappedError(err,
			currOp, "InitInstallationOperation", err.Error())
	}
	return instOp, nil
}

func (c *Controller) compareJobIDs(predecessorMap, predecessorMapNew map[string]*installations.InstallationAndImports) bool {
	if len(predecessorMap) != len(predecessorMapNew) {
		return false
	}

	for name, oldPredecessor := range predecessorMap {
		newPredecessor := predecessorMapNew[name]
		if newPredecessor == nil {
			return false
		}

		if oldPredecessor.GetInstallation().Status.JobID != newPredecessor.GetInstallation().Status.JobID {
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
	phase lsv1alpha1.InstallationPhase, lsError lserrors.LsError, writeID read_write_layer.WriteID,
	reduceLogLevelForConflicts bool) lserrors.LsError {

	op := "setInstallationPhaseAndUpdate"

	logger, ctx := logging.FromContextOrNew(ctx,
		[]interface{}{lc.KeyReconciledResource, client.ObjectKeyFromObject(inst).String()},
		lc.KeyMethod, op)

	inst.Status.LastError = lserrors.TryUpdateLsError(inst.Status.LastError, lsError)

	if inst.Status.LastError != nil {
		lastErr := inst.Status.LastError
		c.EventRecorder().Event(inst, corev1.EventTypeWarning, lastErr.Reason, lastErr.Message)
	}

	if phase != inst.Status.InstallationPhase {
		now := metav1.Now()
		inst.Status.PhaseTransitionTime = &now
	}
	inst.Status.InstallationPhase = phase
	if phase.IsFinal() {
		inst.Status.JobIDFinished = inst.Status.JobID
		inst.Status.TransitionTimes = utils.SetFinishedTransitionTime(inst.Status.TransitionTimes)
	}

	if inst.Status.JobIDFinished == inst.Status.JobID && inst.DeletionTimestamp.IsZero() {
		// The installation is about to finish. Put the names of dependent installations in the status.
		// The dependents will then be triggered in the beginning of the next reconcile event.
		dependents, err := installations.NewInstallationTrigger(c.Client(), inst).DetermineDependents(ctx)
		if err != nil {
			logger.Error(err, "unable to determine successor installations")
			if lsError == nil {
				return lserrors.NewWrappedError(err, op, "DetermineDependents", err.Error())
			}
			return lsError
		}

		inst.Status.DependentsToTrigger = dependents
	}

	err := c.Writer().UpdateInstallationStatus(ctx, writeID, inst)
	if err != nil {
		if inst.Status.InstallationPhase == lsv1alpha1.InstallationPhases.Deleting {
			// recheck if already deleted
			instRecheck := &lsv1alpha1.Installation{}
			errRecheck := read_write_layer.GetInstallation(ctx, c.Client(), kutil.ObjectKey(inst.Name, inst.Namespace), instRecheck)
			if errRecheck != nil && apierrors.IsNotFound(errRecheck) {
				return nil
			}
		}

		// reduceLogLevelForConflicts is set on true, if conflicts might occur, e.g.
		// - when deleting an item a touch operation might be triggered for all siblings to speed up the operation
		if reduceLogLevelForConflicts && apierrors.IsConflict(err) {
			logger.Info("unable to update installation status", err, err.Error())
		} else {
			logger.Error(err, "unable to update installation status")
		}
		if lsError == nil {
			return lserrors.NewWrappedError(err, op, "UpdateInstallationStatus", err.Error())
		}

		return lsError
	}

	return lsError
}
