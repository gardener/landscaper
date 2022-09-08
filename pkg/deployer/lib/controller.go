// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package lib

import (
	"context"
	"fmt"
	"time"

	lsutil "github.com/gardener/landscaper/pkg/utils"

	"github.com/go-logr/logr"
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
	"github.com/gardener/landscaper/pkg/deployer/lib/targetselector"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
	"github.com/gardener/landscaper/pkg/version"
)

// Deployer defines a controller that acts upon deployitems.
type Deployer interface {
	// Reconcile the deployitem.
	Reconcile(ctx context.Context, lsContext *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error
	// Delete the deployitem.
	Delete(ctx context.Context, lsContext *lsv1alpha1.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error
	// Abort the deployitem progress.
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
	con := NewController(lsMgr.GetClient(),
		lsMgr.GetScheme(),
		lsMgr.GetEventRecorderFor(args.Name),
		hostMgr.GetClient(),
		hostMgr.GetScheme(),
		args)

	log = log.Reconciles("", "DeployItem").WithValues(lc.KeyDeployItemType, string(args.Type))

	return builder.ControllerManagedBy(lsMgr).
		For(&lsv1alpha1.DeployItem{}, builder.WithPredicates(NewTypePredicate(args.Type))).
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
}

// NewController creates a new generic deployitem controller.
func NewController(lsClient client.Client,
	lsScheme *runtime.Scheme,
	lsEventRecorder record.EventRecorder,
	hostClient client.Client,
	hostScheme *runtime.Scheme,
	args DeployerArgs) *controller {
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
	}
}

func (c *controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger, ctx := logging.MustStartReconcileFromContext(ctx, req, nil)

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

	if di.Status.GetJobID() != di.Status.JobIDFinished {
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
	return HandleReconcileResult(ctx, err, oldDeployItem, deployItem, c.lsClient, c.lsEventRecorder)
}

func (c *controller) checkTargetResponsibility(ctx context.Context, log logging.Logger, deployItem *lsv1alpha1.DeployItem) (*lsv1alpha1.Target, bool, error) {
	if deployItem.Spec.Target == nil {
		log.Debug("No target defined")
		return nil, true, nil
	}
	log.Debug("Found target. Checking responsibility")
	target := &lsv1alpha1.Target{}
	deployItem.Spec.Target.Namespace = deployItem.Namespace
	if err := c.lsClient.Get(ctx, deployItem.Spec.Target.NamespacedName(), target); err != nil {
		return nil, false, fmt.Errorf("unable to get target for deployitem: %w", err)
	}
	if len(c.targetSelectors) == 0 {
		log.Debug("No target selectors defined")
		return target, true, nil
	}
	matched, err := targetselector.MatchOne(target, c.targetSelectors)
	if err != nil {
		return nil, false, fmt.Errorf("unable to match target selector: %w", err)
	}
	if !matched {
		log.Debug("The deployitem's target has not matched the given target selector",
			"target", target.Name)
		return nil, false, nil
	}
	return target, true, nil
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
	logger, ctx := logging.FromContextOrNew(ctx, nil)
	if lsv1alpha1helper.HasDeleteWithoutUninstallAnnotation(deployItem.ObjectMeta) {
		logger.Info("Deleting deployitem %s without uninstall", deployItem.Name)
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

func (c *controller) Writer() *read_write_layer.Writer {
	return read_write_layer.NewWriter(c.lsClient)
}

func (c *controller) updateDiForNewReconcile(ctx context.Context, di *lsv1alpha1.DeployItem) error {
	di.Status.ObservedGeneration = di.Generation
	now := metav1.Now()
	di.Status.LastReconcileTime = &now
	di.Status.Deployer = c.info
	lsutil.InitErrors(&di.Status)

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
