// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package lib

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrl "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lserrors "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/deployer/lib/extension"
	"github.com/gardener/landscaper/pkg/deployer/lib/targetselector"
	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
	"github.com/gardener/landscaper/pkg/version"
)

// Deployer defines a controller that acts upon deploy items.
type Deployer interface {
	// Reconcile the deploy item.
	Reconcile(ctx context.Context, lsContext *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error
	// Delete the deploy item.
	Delete(ctx context.Context, lsContext *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error
	// ForceReconcile the deploy item.
	// Keep in mind that the force deletion annotation must be removed by the Deployer.
	ForceReconcile(ctx context.Context, lsContext *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error
	// Abort the deploy item progress.
	Abort(ctx context.Context, lsContext *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error
	// ExtensionHooks returns all registered extension hooks.
	ExtensionHooks() extension.ReconcileExtensionHooks
}

// DeployerArgs defines the deployer arguments for the initializing a generic deployer controller.
type DeployerArgs struct {
	Name            string
	Version         string
	Identity        string
	Type            lsv1alpha1.DeployItemType
	Deployer        Deployer
	TargetSelectors []lsv1alpha1.TargetSelector
	Options         ctrl.Options
}

// Default defaults deployer arguments
func (args *DeployerArgs) Default() {
	if len(args.Name) == 0 {
		args.Name = "generic-deployer-library"
	}
	if len(args.Version) == 0 {
		args.Version = version.Get().String()
	}
	if len(args.Identity) == 0 {
		args.Identity = fmt.Sprintf("%s-%d", args.Name, time.Now().UTC().Unix())
	}
}

// Validate validates the provided deployer arguments
func (args DeployerArgs) Validate() error {
	var allErrs []error
	if len(args.Type) == 0 {
		allErrs = append(allErrs, fmt.Errorf("a type must be provided"))
	}
	if args.Deployer == nil {
		allErrs = append(allErrs, fmt.Errorf("a deployer implementation must be provided"))
	}
	return errors.NewAggregate(allErrs)
}

// Add adds a deployer to the given managers using the given args.
func Add(log logging.Logger, lsMgr, hostMgr manager.Manager, args DeployerArgs) error {
	args.Default()
	if err := args.Validate(); err != nil {
		return err
	}
	con := NewController(log,
		lsMgr.GetClient(),
		lsMgr.GetScheme(),
		lsMgr.GetEventRecorderFor(args.Name),
		hostMgr.GetClient(),
		hostMgr.GetScheme(),
		args)

	return builder.ControllerManagedBy(lsMgr).
		For(&lsv1alpha1.DeployItem{}, builder.WithPredicates(NewTypePredicate(args.Type))).
		WithOptions(args.Options).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.Logr() }).
		Complete(con)
}

// controller reconciles deploy items and delegates the business logic to the configured Deployer.
type controller struct {
	log      logging.Logger
	deployer Deployer
	info     lsv1alpha1.DeployerInformation
	// deployerType defines the deployer type the deployer is responsible for.
	deployerType    lsv1alpha1.DeployItemType
	targetSelectors []lsv1alpha1.TargetSelector

	lsClient        client.Client
	lsScheme        *runtime.Scheme
	lsEventRecorder record.EventRecorder
	hostClient      client.Client
	hostScheme      *runtime.Scheme
}

// NewController creates a new generic deployitem controller.
func NewController(log logging.Logger,
	lsClient client.Client,
	lsScheme *runtime.Scheme,
	lsEventRecorder record.EventRecorder,
	hostClient client.Client,
	hostScheme *runtime.Scheme,
	args DeployerArgs) *controller {
	return &controller{
		log:          log,
		deployerType: args.Type,
		deployer:     args.Deployer,
		info: lsv1alpha1.DeployerInformation{
			Identity: args.Identity,
			Name:     args.Name,
			Version:  args.Version,
		},
		targetSelectors: args.TargetSelectors,
		lsClient:        lsClient,
		lsScheme:        lsScheme,
		lsEventRecorder: lsEventRecorder,
		hostClient:      hostClient,
		hostScheme:      hostScheme,
	}
}

// Reconcile implements the reconcile.Reconciler interface that reconciles DeployItems.
func (c *controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	if !utils.IsNewReconcile() {
		return c.reconcileOld(ctx, req)
	} else {
		return c.reconcileNew(ctx, req)
	}
}

func (c *controller) reconcileNew(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := c.log.WithValues("resource", req.NamespacedName)
	logger.Debug("reconcile")

	var err error

	di := &lsv1alpha1.DeployItem{}
	if err := read_write_layer.GetDeployItem(ctx, c.lsClient, req.NamespacedName, di); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Debug(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	c.lsScheme.Default(di)

	target, shouldReconcile, err := c.checkTargetResponsibility(ctx, logger, di)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !shouldReconcile {
		return reconcile.Result{}, nil
	}

	old := di.DeepCopy()

	lsCtx := &lsv1alpha1.Context{}
	// todo: check for real repository context. Maybe overwritten by installation.
	if err := c.lsClient.Get(ctx, kutil.ObjectKey(di.Spec.Context, di.Namespace), lsCtx); err != nil {
		return reconcile.Result{}, fmt.Errorf("unable to get landscaper context: %w", err)
	}

	if di.Status.JobID != di.Status.JobIDFinished {
		if di.Status.DeployItemPhase == lsv1alpha1.DeployItemPhaseSucceeded ||
			di.Status.DeployItemPhase == lsv1alpha1.DeployItemPhaseFailed ||
			di.Status.DeployItemPhase == "" {

			di.Status.Phase = lsv1alpha1.ExecutionPhaseInit
			if di.DeletionTimestamp.IsZero() {
				di.Status.DeployItemPhase = lsv1alpha1.DeployItemPhaseProgressing
			} else {
				di.Status.DeployItemPhase = lsv1alpha1.DeployItemPhaseDeleting
			}

			if err := c.updateDiForNewReconcile(ctx, di); err != nil {
				return reconcile.Result{}, err
			}
		}

		var err lserrors.LsError
		if di.DeletionTimestamp.IsZero() {
			err = c.reconcile(ctx, lsCtx, di, target)
		} else {
			err = c.delete(ctx, lsCtx, di, target)
		}
		return reconcile.Result{}, c.handleReconcileResult(ctx, err, old, di)
	} else {
		return reconcile.Result{}, nil
	}
}

func (c *controller) handleReconcileResult(ctx context.Context, err lserrors.LsError, oldDeployItem, deployItem *lsv1alpha1.DeployItem) error {
	deployItem.Status.LastError = lserrors.TryUpdateLsError(deployItem.Status.LastError, err)

	if deployItem.Status.LastError != nil {
		if lserrors.ContainsAnyErrorCode(deployItem.Status.LastError.Codes, lsv1alpha1.UnrecoverableErrorCodes) {
			deployItem.Status.Phase = lsv1alpha1.ExecutionPhaseFailed
		}

		lastErr := deployItem.Status.LastError
		c.lsEventRecorder.Event(deployItem, corev1.EventTypeWarning, lastErr.Reason, lastErr.Message)
	}

	if deployItem.Status.Phase == lsv1alpha1.ExecutionPhaseFailed {
		deployItem.Status.DeployItemPhase = lsv1alpha1.DeployItemPhaseFailed
	} else if deployItem.Status.Phase == lsv1alpha1.ExecutionPhaseSucceeded {
		deployItem.Status.DeployItemPhase = lsv1alpha1.DeployItemPhaseSucceeded
	}

	if deployItem.Status.DeployItemPhase == lsv1alpha1.DeployItemPhaseSucceeded ||
		deployItem.Status.DeployItemPhase == lsv1alpha1.DeployItemPhaseFailed {
		deployItem.Status.JobIDFinished = deployItem.Status.JobID
	}

	if !reflect.DeepEqual(oldDeployItem.Status, deployItem.Status) {
		if err2 := c.Writer().UpdateDeployItemStatus(ctx, read_write_layer.W000092, deployItem); err2 != nil {
			if !deployItem.DeletionTimestamp.IsZero() {
				// recheck if already deleted
				diRecheck := &lsv1alpha1.DeployItem{}
				errRecheck := read_write_layer.GetDeployItem(ctx, c.lsClient, kutil.ObjectKey(deployItem.Name, deployItem.Namespace), diRecheck)
				if errRecheck != nil && apierrors.IsNotFound(errRecheck) {
					return nil
				}
			}

			if apierrors.IsConflict(err2) { // reduce logging
				c.log.Debug(fmt.Sprintf("unable to update status: %s", err2.Error()))
			} else {
				c.log.Error(err2, "unable to update status")
			}
			if err == nil {
				return err2
			}
		}
	}

	return err
}

// Reconcile implements the reconcile.Reconciler interface that reconciles DeployItems.
func (c *controller) reconcileOld(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := c.log.WithValues("resource", req.NamespacedName)
	logger.Debug("reconcile")
	extensionLogger := logger.WithName("extension")

	var err error
	hookRes := &extension.HookResult{}
	var tmpHookRes *extension.HookResult
	tmpHookRes, lsErr := c.deployer.ExtensionHooks().ExecuteHooks(ctx, extensionLogger, nil, nil, extension.StartHook)
	if lsErr != nil {
		return reconcile.Result{}, lsErr
	}
	hookRes = extension.AggregateHookResults(hookRes, tmpHookRes)
	if hookRes.AbortReconcile {
		return returnAndLogReconcileResult(logger, *hookRes), nil
	}

	di := &lsv1alpha1.DeployItem{}
	if err := read_write_layer.GetDeployItem(ctx, c.lsClient, req.NamespacedName, di); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Debug(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// don't reconcile if ignore annotation is set and installation is not currently running
	if lsv1alpha1helper.HasIgnoreAnnotation(di.ObjectMeta) && lsv1alpha1helper.IsCompletedExecutionPhase(di.Status.Phase) {
		logger.Debug("skipping reconcile due to ignore annotation")
		return reconcile.Result{}, nil
	}

	c.lsScheme.Default(di)

	old := di.DeepCopy()

	target, shouldReconcile, err := c.checkTargetResponsibility(ctx, logger, di)
	if err != nil {
		return reconcile.Result{}, err
	}

	// shouldReconcile can be overwritten by hooks returning a non nil result
	tmpHookRes, lsErr = c.deployer.ExtensionHooks().ExecuteHooks(ctx, extensionLogger, di, target, extension.DuringResponsibilityCheckHook)
	if lsErr != nil {
		return reconcile.Result{}, lsErr
	}
	if tmpHookRes != nil {
		hookRes = extension.AggregateHookResults(hookRes, tmpHookRes)
		shouldReconcile = !hookRes.AbortReconcile
	}
	if !shouldReconcile {
		return returnAndLogReconcileResult(logger, *hookRes), nil
	}

	lsErr = c.removeReconcileTimestampAnnotation(ctx, di)
	if lsErr != nil {
		return reconcile.Result{}, HandleErrorFunc(ctx, lsErr, logger, c.lsClient, c.lsEventRecorder, old, di, false)
	}

	tmpHookRes, lsErr = c.deployer.ExtensionHooks().ExecuteHooks(ctx, extensionLogger, di, target, extension.AfterResponsibilityCheckHook)
	if lsErr != nil {
		return reconcile.Result{}, HandleErrorFunc(ctx, lsErr, logger, c.lsClient, c.lsEventRecorder, old, di, false)
	}
	hookRes = extension.AggregateHookResults(hookRes, tmpHookRes)
	if hookRes.AbortReconcile {
		return returnAndLogReconcileResult(logger, *hookRes), nil
	}

	lsCtx := &lsv1alpha1.Context{}
	// todo: check for real repository context. Maybe overwritten by installation.
	if err := c.lsClient.Get(ctx, kutil.ObjectKey(di.Spec.Context, di.Namespace), lsCtx); err != nil {
		return reconcile.Result{}, fmt.Errorf("unable to get landscaper context: %w", err)
	}

	logger.Debug("check deploy item reconciliation")
	if err := HandleAnnotationsAndGeneration(ctx, logger, c.lsClient, di, c.info); err != nil {
		return reconcile.Result{}, err
	}

	shouldReconcile = ShouldReconcile(di)
	tmpHookRes, lsErr = c.deployer.ExtensionHooks().ExecuteHooks(ctx, extensionLogger, di, target, extension.ShouldReconcileHook)
	if lsErr != nil {
		return reconcile.Result{}, HandleErrorFunc(ctx, lsErr, logger, c.lsClient, c.lsEventRecorder, old, di, false)
	}
	hookRes = extension.AggregateHookResults(hookRes, tmpHookRes)
	if !shouldReconcile {
		if tmpHookRes != nil && !tmpHookRes.AbortReconcile {
			// if ShouldReconcile returned false but this was overwritten by the extension hooks, we need to call PrepareReconcile,
			// as this has not yet been done by HandleAnnotationsAndGeneration
			logger.Debug("reconcile required by extension hook")
			if err := PrepareReconcile(ctx, logger, c.lsClient, di, c.info); err != nil {
				return reconcile.Result{}, err
			}
		} else {
			// neither the default logic nor the extension hooks require a reconcile
			c.log.Debug("aborting reconcile", "phase", di.Status.Phase)
			return returnAndLogReconcileResult(logger, *hookRes), nil
		}
	}
	logger.Info("reconcile deploy item")
	// reset AbortReconcile, since it could be 'true' at this point, which would wrongly cause an abort after the next hook
	hookRes.AbortReconcile = false

	tmpHookRes, lsErr = c.deployer.ExtensionHooks().ExecuteHooks(ctx, extensionLogger, di, target, extension.BeforeAnyReconcileHook)
	if lsErr != nil {
		return reconcile.Result{}, HandleErrorFunc(ctx, lsErr, logger, c.lsClient, c.lsEventRecorder, old, di, false)
	}
	hookRes = extension.AggregateHookResults(hookRes, tmpHookRes)
	if hookRes.AbortReconcile {
		return returnAndLogReconcileResult(logger, *hookRes), nil
	}

	if !di.DeletionTimestamp.IsZero() {
		logger.Debug("handle deploy item deletion")
		tmpHookRes, lsErr = c.deployer.ExtensionHooks().ExecuteHooks(ctx, extensionLogger, di, target, extension.BeforeDeleteHook)
		if lsErr != nil {
			return reconcile.Result{}, HandleErrorFunc(ctx, lsErr, logger, c.lsClient, c.lsEventRecorder, old, di, false)
		}
		hookRes = extension.AggregateHookResults(hookRes, tmpHookRes)
		if hookRes.AbortReconcile {
			return returnAndLogReconcileResult(logger, *hookRes), nil
		}
		if err := HandleErrorFunc(ctx, c.delete(ctx, lsCtx, di, target), logger, c.lsClient, c.lsEventRecorder, old, di, true); err != nil {
			return reconcile.Result{}, err
		}
	} else {
		switch lsv1alpha1.Operation(lsv1alpha1helper.GetOperation(di.ObjectMeta)) {
		case lsv1alpha1.AbortOperation:
			logger.Debug("handle deploy item abort")
			tmpHookRes, lsErr = c.deployer.ExtensionHooks().ExecuteHooks(ctx, extensionLogger, di, target, extension.BeforeAbortHook)
			if lsErr != nil {
				return reconcile.Result{}, HandleErrorFunc(ctx, lsErr, logger, c.lsClient, c.lsEventRecorder, old, di, false)
			}
			hookRes = extension.AggregateHookResults(hookRes, tmpHookRes)
			if hookRes.AbortReconcile {
				return returnAndLogReconcileResult(logger, *hookRes), nil
			}
			err = c.deployer.Abort(ctx, lsCtx, di, target)
			if err := HandleErrorFunc(ctx, lserrors.BuildLsErrorOrNil(err, "Reconcile", "Abort", "abort"),
				logger, c.lsClient, c.lsEventRecorder, old, di, false); err != nil {
				return reconcile.Result{}, err
			}
		case lsv1alpha1.ForceReconcileOperation:
			logger.Debug("handle deploy item force-reconcile")
			tmpHookRes, lsErr = c.deployer.ExtensionHooks().ExecuteHooks(ctx, extensionLogger, di, target, extension.BeforeForceReconcileHook)
			if lsErr != nil {
				return reconcile.Result{}, HandleErrorFunc(ctx, lsErr, logger, c.lsClient, c.lsEventRecorder, old, di, false)
			}
			hookRes = extension.AggregateHookResults(hookRes, tmpHookRes)
			if hookRes.AbortReconcile {
				return returnAndLogReconcileResult(logger, *hookRes), nil
			}
			logger.Debug("removing reconcile annotation")
			delete(di.ObjectMeta.Annotations, lsv1alpha1.OperationAnnotation)
			if err := c.Writer().UpdateDeployItem(ctx, read_write_layer.W000040, di); err != nil {
				return reconcile.Result{}, err
			}

			err = c.deployer.ForceReconcile(ctx, lsCtx, di, target)
			if err := HandleErrorFunc(ctx, lserrors.BuildLsErrorOrNil(err, "Reconcile", "ForceReconcile", "force,reconcile"),
				logger, c.lsClient, c.lsEventRecorder, old, di, false); err != nil {
				return reconcile.Result{}, err
			}
		default:
			// default reconcile
			logger.Debug("handle deploy item reconcile")
			tmpHookRes, lsErr = c.deployer.ExtensionHooks().ExecuteHooks(ctx, extensionLogger, di, target, extension.BeforeReconcileHook)
			if lsErr != nil {
				return reconcile.Result{}, HandleErrorFunc(ctx, lsErr, logger, c.lsClient, c.lsEventRecorder, old, di, false)
			}
			hookRes = extension.AggregateHookResults(hookRes, tmpHookRes)
			if hookRes.AbortReconcile {
				return returnAndLogReconcileResult(logger, *hookRes), nil
			}

			if err := HandleErrorFunc(ctx, c.reconcile(ctx, lsCtx, di, target), logger, c.lsClient, c.lsEventRecorder, old, di, false); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	tmpHookRes, lsErr = c.deployer.ExtensionHooks().ExecuteHooks(ctx, extensionLogger, di, target, extension.EndHook)
	if lsErr != nil {
		return reconcile.Result{}, HandleErrorFunc(ctx, lsErr, logger, c.lsClient, c.lsEventRecorder, old, di, false)
	}
	hookRes = extension.AggregateHookResults(hookRes, tmpHookRes)
	return returnAndLogReconcileResult(logger, *hookRes), nil
}

func (c *controller) checkTargetResponsibility(ctx context.Context, log logging.Logger, deployItem *lsv1alpha1.DeployItem) (*lsv1alpha1.Target, bool, error) {
	if deployItem.Spec.Target == nil {
		log.Debug("no target defined")
		return nil, true, nil
	}
	log.Debug("Found target. Checking responsibility")
	target := &lsv1alpha1.Target{}
	deployItem.Spec.Target.Namespace = deployItem.Namespace
	if err := c.lsClient.Get(ctx, deployItem.Spec.Target.NamespacedName(), target); err != nil {
		return nil, false, fmt.Errorf("unable to get target for deploy item: %w", err)
	}
	if len(c.targetSelectors) == 0 {
		log.Debug("no target selectors defined")
		return target, true, nil
	}
	matched, err := targetselector.MatchOne(target, c.targetSelectors)
	if err != nil {
		return nil, false, fmt.Errorf("unable to match target selector: %w", err)
	}
	if !matched {
		log.Debug("the deploy item's target has not matched the given target selector",
			"target", target.Name)
		return nil, false, nil
	}
	return target, true, nil
}

func returnAndLogReconcileResult(logger logging.Logger, result extension.HookResult) reconcile.Result {
	if result.AbortReconcile {
		logger.Debug("deploy item reconcile has been aborted")
	}
	if result.ReconcileResult.RequeueAfter != 0 {
		logger.Debug("deploy item will be requeued", "duration", result.ReconcileResult.RequeueAfter.String())
	} else if result.ReconcileResult.Requeue {
		logger.Debug("deploy item will be requeued immediately")
	} else {
		logger.Debug("deploy item will not be requeued")
	}
	return result.ReconcileResult
}

func (c *controller) reconcile(ctx context.Context, lsCtx *lsv1alpha1.Context, deployItem *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) lserrors.LsError {
	if !controllerutil.ContainsFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer)
		if err := c.Writer().UpdateDeployItem(ctx, read_write_layer.W000050, deployItem); err != nil {
			return lserrors.NewWrappedError(err,
				"Reconcile", "AddFinalizer", err.Error())
		}
	}

	err := c.deployer.Reconcile(ctx, lsCtx, deployItem, target)
	return lserrors.BuildLsErrorOrNil(err, "reconcile", "Reconcile")
}

func (c *controller) delete(ctx context.Context, lsCtx *lsv1alpha1.Context, deployItem *lsv1alpha1.DeployItem,
	target *lsv1alpha1.Target) lserrors.LsError {
	if lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(deployItem.ObjectMeta) {
		c.log.Info("Deleting deploy item %s without uninstall", deployItem.Name)
	} else {
		if err := c.deployer.Delete(ctx, lsCtx, deployItem, target); err != nil {
			return lserrors.BuildLsError(err, "delete", "DeleteWithUninstall", err.Error())
		}
	}

	if controllerutil.ContainsFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.RemoveFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer)
		if err := c.Writer().UpdateDeployItem(ctx, read_write_layer.W000037, deployItem); err != nil {
			return lserrors.NewWrappedError(err,
				"Reconcile", "RemoveFinalizer", err.Error())
		}
	}
	return nil
}

func (c *controller) removeReconcileTimestampAnnotation(ctx context.Context, deployItem *lsv1alpha1.DeployItem) lserrors.LsError {
	if metav1.HasAnnotation(deployItem.ObjectMeta, lsv1alpha1.ReconcileTimestampAnnotation) {
		delete(deployItem.ObjectMeta.Annotations, lsv1alpha1.ReconcileTimestampAnnotation)

		if err := c.Writer().UpdateDeployItem(ctx, read_write_layer.W000076, deployItem); err != nil {
			return lserrors.BuildLsError(err, "RemoveReconcileTimestampAnnotation", "UpdateMetadata", err.Error())
		}
	}

	return nil
}

func (c *controller) Writer() *read_write_layer.Writer {
	return read_write_layer.NewWriter(c.lsClient)
}

func (c *controller) updateDiForNewReconcile(ctx context.Context, di *lsv1alpha1.DeployItem) error {
	di.Status.ObservedGeneration = di.Generation
	now := metav1.Now()
	di.Status.LastReconcileTime = &now
	di.Status.Deployer = c.info

	if err := c.Writer().UpdateDeployItemStatus(ctx, read_write_layer.W000004, di); err != nil {
		return err
	}

	return nil
}

// typePredicate is a predicate definition that does only react on deployitem of the specific type.
type typePredicate struct {
	Type lsv1alpha1.DeployItemType
}

func NewTypePredicate(dtype lsv1alpha1.DeployItemType) predicate.Predicate {
	return typePredicate{
		Type: dtype,
	}
}

func (p typePredicate) handleObj(obj client.Object) bool {
	di, ok := obj.(*lsv1alpha1.DeployItem)
	if !ok {
		return false
	}
	return di.Spec.Type == p.Type
}

func (p typePredicate) Create(event event.CreateEvent) bool {
	return p.handleObj(event.Object)
}

func (p typePredicate) Delete(event event.DeleteEvent) bool {
	return p.handleObj(event.Object)
}

func (p typePredicate) Update(event event.UpdateEvent) bool {
	return p.handleObj(event.ObjectNew)
}

func (p typePredicate) Generic(event event.GenericEvent) bool {
	return p.handleObj(event.Object)
}

var _ predicate.Predicate = typePredicate{}
