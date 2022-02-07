// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package lib

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"

	"github.com/gardener/landscaper/pkg/version"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/pkg/deployer/lib/extension"
	"github.com/gardener/landscaper/pkg/deployer/lib/targetselector"
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
func Add(log logr.Logger, lsMgr, hostMgr manager.Manager, args DeployerArgs) error {
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
		WithLogger(log).
		Complete(con)
}

// controller reconciles deploy items and delegates the business logic to the configured Deployer.
type controller struct {
	log      logr.Logger
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
func NewController(log logr.Logger,
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
	logger := c.log.WithValues("resource", req.NamespacedName)
	logger.V(7).Info("reconcile")
	extensionLogger := logger.WithName("extension")

	var err error
	hookRes := &extension.HookResult{}
	var tmpHookRes *extension.HookResult
	tmpHookRes, err = c.deployer.ExtensionHooks().ExecuteHooks(ctx, extensionLogger, nil, nil, extension.StartHook)
	if err != nil {
		return reconcile.Result{}, err
	}
	hookRes = extension.AggregateHookResults(hookRes, tmpHookRes)
	if hookRes.AbortReconcile {
		return returnAndLogReconcileResult(logger, *hookRes), nil
	}

	di := &lsv1alpha1.DeployItem{}
	if err := c.lsClient.Get(ctx, req.NamespacedName, di); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// don't reconcile if ignore annotation is set and installation is not currently running
	if lsv1alpha1helper.HasIgnoreAnnotation(di.ObjectMeta) && lsv1alpha1helper.IsCompletedExecutionPhase(di.Status.Phase) {
		logger.V(7).Info("skipping reconcile due to ignore annotation")
		return reconcile.Result{}, nil
	}

	c.lsScheme.Default(di)

	errHdl := HandleErrorFunc(logger, c.lsClient, c.lsEventRecorder, di)

	target, shouldReconcile, err := c.checkTargetResponsibility(ctx, logger, di)
	if err != nil {
		return reconcile.Result{}, err
	}
	tmpHookRes, err = c.deployer.ExtensionHooks().ExecuteHooks(ctx, extensionLogger, di, target, extension.DuringResponsibilityCheckHook)
	if err != nil {
		return reconcile.Result{}, err
	}
	if tmpHookRes != nil {
		hookRes = extension.AggregateHookResults(hookRes, tmpHookRes)
		shouldReconcile = !hookRes.AbortReconcile
	}
	if !shouldReconcile {
		return returnAndLogReconcileResult(logger, *hookRes), nil
	}

	tmpHookRes, err = c.deployer.ExtensionHooks().ExecuteHooks(ctx, extensionLogger, di, target, extension.AfterResponsibilityCheckHook)
	if err != nil {
		return reconcile.Result{}, errHdl(ctx, err)
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

	logger.V(3).Info("check deploy item reconciliation")
	if err := HandleAnnotationsAndGeneration(ctx, logger, c.lsClient, di, c.info); err != nil {
		return reconcile.Result{}, err
	}

	shouldReconcile = ShouldReconcile(di)
	tmpHookRes, err = c.deployer.ExtensionHooks().ExecuteHooks(ctx, extensionLogger, di, target, extension.ShouldReconcileHook)
	if err != nil {
		return reconcile.Result{}, errHdl(ctx, err)
	}
	hookRes = extension.AggregateHookResults(hookRes, tmpHookRes)
	if !shouldReconcile {
		if tmpHookRes != nil && !tmpHookRes.AbortReconcile {
			// if ShouldReconcile returned false but this was overwritten by the extension hooks, we need to call PrepareReconcile,
			// as this has not yet been done by HandleAnnotationsAndGeneration
			logger.V(5).Info("reconcile required by extension hook")
			if err := PrepareReconcile(ctx, logger, c.lsClient, di, c.info); err != nil {
				return reconcile.Result{}, err
			}
		} else {
			// neither the default logic nor the extension hooks require a reconcile
			c.log.V(5).Info("aborting reconcile", "phase", di.Status.Phase)
			return returnAndLogReconcileResult(logger, *hookRes), nil
		}
	}
	logger.Info("reconcile deploy item")
	// reset AbortReconcile, since it could be 'true' at this point, which would wrongly cause an abort after the next hook
	hookRes.AbortReconcile = false

	tmpHookRes, err = c.deployer.ExtensionHooks().ExecuteHooks(ctx, extensionLogger, di, target, extension.BeforeAnyReconcileHook)
	if err != nil {
		return reconcile.Result{}, errHdl(ctx, err)
	}
	hookRes = extension.AggregateHookResults(hookRes, tmpHookRes)
	if hookRes.AbortReconcile {
		return returnAndLogReconcileResult(logger, *hookRes), nil
	}

	switch lsv1alpha1.Operation(lsv1alpha1helper.GetOperation(di.ObjectMeta)) {
	case lsv1alpha1.AbortOperation:
		logger.V(5).Info("handle deploy item abort")
		tmpHookRes, err = c.deployer.ExtensionHooks().ExecuteHooks(ctx, extensionLogger, di, target, extension.BeforeAbortHook)
		if err != nil {
			return reconcile.Result{}, errHdl(ctx, err)
		}
		hookRes = extension.AggregateHookResults(hookRes, tmpHookRes)
		if hookRes.AbortReconcile {
			return returnAndLogReconcileResult(logger, *hookRes), nil
		}
		if err := errHdl(ctx, c.deployer.Abort(ctx, lsCtx, di, target)); err != nil {
			return reconcile.Result{}, err
		}
	case lsv1alpha1.ForceReconcileOperation:
		logger.V(5).Info("handle deploy item force-reconcile")
		tmpHookRes, err = c.deployer.ExtensionHooks().ExecuteHooks(ctx, extensionLogger, di, target, extension.BeforeForceReconcileHook)
		if err != nil {
			return reconcile.Result{}, errHdl(ctx, err)
		}
		hookRes = extension.AggregateHookResults(hookRes, tmpHookRes)
		if hookRes.AbortReconcile {
			return returnAndLogReconcileResult(logger, *hookRes), nil
		}
		if err := errHdl(ctx, c.deployer.ForceReconcile(ctx, lsCtx, di, target)); err != nil {
			return reconcile.Result{}, err
		}
	default:

		if !di.DeletionTimestamp.IsZero() {
			logger.V(5).Info("handle deploy item deletion")
			tmpHookRes, err = c.deployer.ExtensionHooks().ExecuteHooks(ctx, extensionLogger, di, target, extension.BeforeDeleteHook)
			if err != nil {
				return reconcile.Result{}, errHdl(ctx, err)
			}
			hookRes = extension.AggregateHookResults(hookRes, tmpHookRes)
			if hookRes.AbortReconcile {
				return returnAndLogReconcileResult(logger, *hookRes), nil
			}
			if err := errHdl(ctx, c.delete(ctx, lsCtx, di, target)); err != nil {
				return reconcile.Result{}, err
			}
		} else {
			// default reconcile
			logger.V(7).Info("handle deploy item reconcile")
			tmpHookRes, err = c.deployer.ExtensionHooks().ExecuteHooks(ctx, extensionLogger, di, target, extension.BeforeReconcileHook)
			if err != nil {
				return reconcile.Result{}, errHdl(ctx, err)
			}
			hookRes = extension.AggregateHookResults(hookRes, tmpHookRes)
			if hookRes.AbortReconcile {
				return returnAndLogReconcileResult(logger, *hookRes), nil
			}
			if err := errHdl(ctx, c.reconcile(ctx, lsCtx, di, target)); err != nil {
				return reconcile.Result{}, err
			}
		}

	}

	tmpHookRes, err = c.deployer.ExtensionHooks().ExecuteHooks(ctx, extensionLogger, di, target, extension.EndHook)
	if err != nil {
		return reconcile.Result{}, errHdl(ctx, err)
	}
	hookRes = extension.AggregateHookResults(hookRes, tmpHookRes)
	return returnAndLogReconcileResult(logger, *hookRes), nil
}

func (c *controller) checkTargetResponsibility(ctx context.Context, log logr.Logger, deployItem *lsv1alpha1.DeployItem) (*lsv1alpha1.Target, bool, error) {
	if deployItem.Spec.Target == nil {
		log.V(9).Info("no target defined")
		return nil, true, nil
	}
	log.V(7).Info("Found target. Checking responsibility")
	target := &lsv1alpha1.Target{}
	deployItem.Spec.Target.Namespace = deployItem.Namespace
	if err := c.lsClient.Get(ctx, deployItem.Spec.Target.NamespacedName(), target); err != nil {
		return nil, false, fmt.Errorf("unable to get target for deploy item: %w", err)
	}
	if len(c.targetSelectors) == 0 {
		log.V(9).Info("no target selectors defined")
		return target, true, nil
	}
	matched, err := targetselector.MatchOne(target, c.targetSelectors)
	if err != nil {
		return nil, false, fmt.Errorf("unable to match target selector: %w", err)
	}
	if !matched {
		log.V(5).Info("the deploy item's target has not matched the given target selector",
			"target", target.Name)
		return nil, false, nil
	}
	return target, true, nil
}

func returnAndLogReconcileResult(logger logr.Logger, result extension.HookResult) reconcile.Result {
	if result.AbortReconcile {
		logger.V(7).Info("deploy item reconcile has been aborted")
	}
	if result.ReconcileResult.RequeueAfter != 0 {
		logger.V(5).Info("deploy item will be requeued", "duration", result.ReconcileResult.RequeueAfter.String())
	} else if result.ReconcileResult.Requeue {
		logger.V(5).Info("deploy item will be requeued immediately")
	} else {
		logger.V(7).Info("deploy item will not be requeued")
	}
	return result.ReconcileResult
}

func (c *controller) reconcile(ctx context.Context, lsCtx *lsv1alpha1.Context, deployItem *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error {
	if !controllerutil.ContainsFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer)
		if err := c.lsClient.Update(ctx, deployItem); err != nil {
			return lserrors.NewWrappedError(err,
				"Reconcile", "AddFinalizer", err.Error())
		}
	}

	return c.deployer.Reconcile(ctx, lsCtx, deployItem, target)
}

func (c *controller) delete(ctx context.Context, lsCtx *lsv1alpha1.Context, deployItem *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error {
	if err := c.deployer.Delete(ctx, lsCtx, deployItem, target); err != nil {
		return err
	}
	if controllerutil.ContainsFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.RemoveFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer)
		if err := c.lsClient.Update(ctx, deployItem); err != nil {
			return lserrors.NewWrappedError(err,
				"Reconcile", "RemoveFinalizer", err.Error())
		}
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
