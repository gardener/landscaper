// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package lib

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
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
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/deployer/lib/extension"
	lsutil "github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/lock"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
	"github.com/gardener/landscaper/pkg/version"
)

// Deployer defines a controller that acts upon deployitems.
type Deployer interface {
	// Reconcile the deployitem.
	Reconcile(ctx context.Context, lsContext *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.ResolvedTarget) error
	// Delete the deployitem.
	Delete(ctx context.Context, lsContext *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.ResolvedTarget) error
	// Abort the deployitem progress.
	Abort(ctx context.Context, lsContext *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.ResolvedTarget) error
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
func Add(log logging.Logger, lsMgr, hostMgr manager.Manager, args DeployerArgs, maxNumberOfWorkers int, lockingEnabled bool, callerName string) error {
	args.Default()
	if err := args.Validate(); err != nil {
		return err
	}
	con := NewController(lsMgr.GetClient(),
		lsMgr.GetScheme(),
		lsMgr.GetEventRecorderFor(args.Name),
		hostMgr.GetClient(),
		hostMgr.GetScheme(),
		args,
		maxNumberOfWorkers,
		lockingEnabled,
		callerName)

	log = log.Reconciles("", "DeployItem").WithValues(lc.KeyDeployItemType, string(args.Type))

	return builder.ControllerManagedBy(lsMgr).
		For(&lsv1alpha1.DeployItem{}, builder.WithPredicates(NewTypePredicate(args.Type)), builder.OnlyMetadata).
		WithOptions(args.Options).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.Logr() }).
		Complete(con)
}

// controller reconciles deployitems and delegates the business logic to the configured Deployer.
type controller struct {
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

	workerCounter  *lsutil.WorkerCounter
	lockingEnabled bool
	callerName     string
}

// NewController creates a new generic deployitem controller.
func NewController(lsClient client.Client,
	lsScheme *runtime.Scheme,
	lsEventRecorder record.EventRecorder,
	hostClient client.Client,
	hostScheme *runtime.Scheme,
	args DeployerArgs,
	maxNumberOfWorkers int,
	lockingEnabled bool,
	callerName string) *controller {

	wc := lsutil.NewWorkerCounter(maxNumberOfWorkers)

	return &controller{
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
		workerCounter:   wc,
		lockingEnabled:  lockingEnabled,
		callerName:      callerName,
	}
}

func (c *controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger, ctx := logging.MustStartReconcileFromContext(ctx, req, nil)

	c.workerCounter.EnterWithLog(logger, 70, c.callerName)
	defer c.workerCounter.Exit()

	metadata := lsutil.EmptyDeployItemMetadata()
	if err := c.lsClient.Get(ctx, req.NamespacedName, metadata); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Debug(err.Error())
			return reconcile.Result{}, nil
		}
		return lsutil.LogHelper{}.LogStandardErrorAndGetReconcileResult(ctx, err)
	}

	// this check is only for compatibility reasons
	rt, reponsible, err := CheckResponsibility(ctx, c.lsClient, metadata, c.deployerType, c.targetSelectors)
	if err != nil {
		return lsutil.LogHelper{}.LogErrorAndGetReconcileResult(ctx, err)
	}

	if !reponsible {
		return reconcile.Result{}, nil
	}

	if c.lockingEnabled {
		locker := lock.NewLocker(c.lsClient, c.hostClient, c.callerName)
		syncObject, err := locker.LockDI(ctx, metadata)
		if err != nil {
			return lsutil.LogHelper{}.LogErrorAndGetReconcileResult(ctx, err)
		}

		if syncObject == nil {
			return locker.NotLockedResult()
		}

		defer func() {
			locker.Unlock(ctx, syncObject)
		}()
	}

	return c.reconcilePrivate(ctx, metadata, rt)
}

func (c *controller) reconcilePrivate(ctx context.Context, metadata *metav1.PartialObjectMetadata,
	rt *lsv1alpha1.ResolvedTarget) (reconcile.Result, error) {

	logger, ctx := logging.FromContextOrNew(ctx, nil)
	di := &lsv1alpha1.DeployItem{}
	if err := read_write_layer.GetDeployItem(ctx, c.lsClient, client.ObjectKeyFromObject(metadata), di); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Debug(err.Error())
			return reconcile.Result{}, nil
		}
		return lsutil.LogHelper{}.LogStandardErrorAndGetReconcileResult(ctx, err)
	}

	// do we really need to check if metadata and di have the same guid?
	if metadata.UID != di.UID {
		err := lserrors.NewError("Reconcile", "differentUIDs", "different UIDs")
		return lsutil.LogHelper{}.LogErrorAndGetReconcileResult(ctx, err)
	}

	c.lsScheme.Default(di)

	old := di.DeepCopy()

	hasTestReconcileAnnotation := lsv1alpha1helper.HasOperation(di.ObjectMeta, lsv1alpha1.TestReconcileOperation)

	if !hasTestReconcileAnnotation && di.Status.GetJobID() == di.Status.JobIDFinished {
		logger.Info("deploy item not reconciled because no new job ID or test reconcile annotation")
		return reconcile.Result{}, nil
	}

	if hasTestReconcileAnnotation {
		if err := c.removeTestReconcileAnnotation(ctx, di); err != nil {
			return lsutil.LogHelper{}.LogStandardErrorAndGetReconcileResult(ctx, err)
		}

		logger.Info("generating a new jobID, because of a test-reconcile annotation")
		di.Status.SetJobID(uuid.New().String())
		di.Status.TransitionTimes = lsutil.NewTransitionTimes()
	}

	if di.Status.Phase.IsFinal() || di.Status.Phase.IsEmpty() {
		// The deployitem has a new jobID, but the phase is still finished from before
		if di.DeletionTimestamp.IsZero() {
			if di.Spec.UpdateOnChangeOnly &&
				di.GetGeneration() == di.Status.ObservedGeneration &&
				di.Status.Phase == lsv1alpha1.DeployItemPhases.Succeeded &&
				!hasTestReconcileAnnotation {

				// deployitem is unchanged and succeeded, and no reconcile desired in this case
				c.initStatus(ctx, di)
				err := c.handleReconcileResult(ctx, nil, old, di)
				return lsutil.LogHelper{}.LogStandardErrorAndGetReconcileResult(ctx, err)
			}

			// initialize deployitem for reconcile
			logger.Debug("Setting deployitem to phase 'Init'", "updateOnChangeOnly", di.Spec.UpdateOnChangeOnly, lc.KeyGeneration, di.GetGeneration(), lc.KeyObservedGeneration, di.Status.ObservedGeneration, lc.KeyDeployItemPhase, di.Status.Phase)
			di.Status.Phase = lsv1alpha1.DeployItemPhases.Init
			if err := c.initAndUpdateStatus(ctx, di); err != nil {
				return lsutil.LogHelper{}.LogStandardErrorAndGetReconcileResult(ctx, err)
			}
		} else {
			// initialize deployitem for delete
			di.Status.Phase = lsv1alpha1.DeployItemPhases.InitDelete
			if err := c.initAndUpdateStatus(ctx, di); err != nil {
				return lsutil.LogHelper{}.LogStandardErrorAndGetReconcileResult(ctx, err)
			}
		}
	}

	// Deployitem has been initialized, proceed with reconcile/delete

	if di.DeletionTimestamp.IsZero() {
		lsError := c.reconcile(ctx, di, rt)
		_ = c.handleReconcileResult(ctx, lsError, old, di)
		return c.buildResult(di.Status.Phase)

	} else {
		lsError := c.delete(ctx, di, rt)
		_ = c.handleReconcileResult(ctx, lsError, old, di)
		return c.buildResult(di.Status.Phase)
	}
}

func (c *controller) handleReconcileResult(ctx context.Context, err lserrors.LsError, oldDeployItem, deployItem *lsv1alpha1.DeployItem) error {
	return HandleReconcileResult(ctx, err, oldDeployItem, deployItem, c.lsClient, c.lsEventRecorder)
}

func (c *controller) buildResult(phase lsv1alpha1.DeployItemPhase) (reconcile.Result, error) {
	if phase.IsFinal() {
		return reconcile.Result{}, nil
	} else {
		// Init, Progressing, or Deleting
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}
}

func (c *controller) getContext(ctx context.Context, deployItem *lsv1alpha1.DeployItem,
	operation string) (*lsv1alpha1.Context, lserrors.LsError) {

	contextName := deployItem.Spec.Context
	if len(contextName) == 0 {
		contextName = lsv1alpha1.DefaultContextName
	}

	lsCtx := &lsv1alpha1.Context{}
	if err := c.lsClient.Get(ctx, kutil.ObjectKey(contextName, deployItem.Namespace), lsCtx); err != nil {
		return nil, lserrors.NewWrappedError(err, operation, "GetLandscaperContext", err.Error())
	}

	return lsCtx, nil
}

func (c *controller) reconcile(ctx context.Context, deployItem *lsv1alpha1.DeployItem,
	rt *lsv1alpha1.ResolvedTarget) lserrors.LsError {

	operation := "reconcile"

	if !controllerutil.ContainsFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer)
		if err := c.Writer().UpdateDeployItem(ctx, read_write_layer.W000050, deployItem); err != nil {
			return lserrors.NewWrappedError(err, operation, "AddFinalizer", err.Error())
		}
	}

	lsCtx, lsErr := c.getContext(ctx, deployItem, operation)
	if lsErr != nil {
		return lsErr
	}

	err := c.deployer.Reconcile(ctx, lsCtx, deployItem, rt)
	return lserrors.BuildLsErrorOrNil(err, operation, "Reconcile")
}

func (c *controller) delete(ctx context.Context, deployItem *lsv1alpha1.DeployItem,
	rt *lsv1alpha1.ResolvedTarget) lserrors.LsError {

	logger, ctx := logging.FromContextOrNew(ctx, nil)
	operation := "delete"

	if lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(deployItem.ObjectMeta) {
		// this case is not required anymore because those items are removed by the execution controller
		// but for security reasons not removed
		logger.Info("Deleting deployitem %s without uninstall", deployItem.Name)
	} else {
		lsCtx, lsErr := c.getContext(ctx, deployItem, operation)
		if lsErr != nil {
			return lsErr
		}

		if err := c.deployer.Delete(ctx, lsCtx, deployItem, rt); err != nil {
			return lserrors.BuildLsError(err, operation, "DeleteWithUninstall", err.Error())
		}
	}

	if controllerutil.ContainsFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.RemoveFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer)
		if err := c.Writer().UpdateDeployItem(ctx, read_write_layer.W000037, deployItem); err != nil {
			return lserrors.NewWrappedError(err, operation, "RemoveFinalizer", err.Error())
		}
	}
	return nil
}

func (c *controller) Writer() *read_write_layer.Writer {
	return read_write_layer.NewWriter(c.lsClient)
}

func (c *controller) initAndUpdateStatus(ctx context.Context, di *lsv1alpha1.DeployItem) error {
	c.initStatus(ctx, di)

	if err := c.Writer().UpdateDeployItemStatus(ctx, read_write_layer.W000004, di); err != nil {
		return err
	}

	return nil
}

func (c *controller) initStatus(ctx context.Context, di *lsv1alpha1.DeployItem) {
	di.Status.ObservedGeneration = di.Generation
	di.Status.TransitionTimes = lsutil.SetInitTransitionTime(di.Status.TransitionTimes)
	now := metav1.Now()
	di.Status.LastReconcileTime = &now
	di.Status.Deployer = c.info
	lsutil.InitErrors(&di.Status)
}

func (c *controller) removeTestReconcileAnnotation(ctx context.Context, di *lsv1alpha1.DeployItem) lserrors.LsError {
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyReconciledResource, client.ObjectKeyFromObject(di).String()})

	logger.Info("remove test-reconcile annotation")
	delete(di.Annotations, lsv1alpha1.OperationAnnotation)
	if err := c.Writer().UpdateDeployItem(ctx, read_write_layer.W000149, di); client.IgnoreNotFound(err) != nil {
		return lserrors.NewWrappedError(err, "RemoveTestReconcileAnnotation", "UpdateDeployItem", err.Error())
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
	deployerType, found := obj.GetAnnotations()[lsv1alpha1.DeployerTypeAnnotation]
	if !found {
		return true
	}

	return deployerType == string(p.Type)
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
