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
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/pkg/version"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/deployer/lib/targetselector"
)

// Deployer defines a controller that acts upon deploy items.
type Deployer interface {
	// Reconcile the deploy item.
	Reconcile(ctx context.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error
	// Delete the deploy item.
	Delete(ctx context.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error
	// ForceReconcile the deploy item.
	// Keep in mind that the force deletion annotation must be removed by the Deployer.
	ForceReconcile(ctx context.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error
	// Abort the deploy item progress.
	Abort(ctx context.Context, di *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error
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
	con := NewController(log, lsMgr.GetClient(), lsMgr.GetScheme(), hostMgr.GetClient(), hostMgr.GetScheme(), args)

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

	lsClient   client.Client
	lsScheme   *runtime.Scheme
	hostClient client.Client
	hostScheme *runtime.Scheme
}

// NewController creates a new generic deployitem controller.
func NewController(log logr.Logger,
	lsClient client.Client,
	lsScheme *runtime.Scheme,
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
		hostClient:      hostClient,
		hostScheme:      hostScheme,
	}
}

// Reconcile implements the reconcile.Reconciler interface that reconciles DeployItems.
func (c *controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := c.log.WithValues("resource", req.NamespacedName)
	logger.V(7).Info("reconcile")

	deployItem := &lsv1alpha1.DeployItem{}
	if err := c.lsClient.Get(ctx, req.NamespacedName, deployItem); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	var target *lsv1alpha1.Target
	if deployItem.Spec.Target != nil {
		target = &lsv1alpha1.Target{}
		if err := c.lsClient.Get(ctx, deployItem.Spec.Target.NamespacedName(), target); err != nil {
			return reconcile.Result{}, fmt.Errorf("unable to get target for deploy item: %w", err)
		}
		if len(c.targetSelectors) != 0 {
			matched, err := targetselector.MatchOne(target, c.targetSelectors)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("unable to match target selector: %w", err)
			}
			if !matched {
				logger.V(5).Info("the deploy item's target has not matched the given target selector",
					"target", target.Name)
				return reconcile.Result{}, nil
			}
		}
	}

	logger.V(3).Info("check deploy item reconciliation")
	err := HandleAnnotationsAndGeneration(ctx, logger, c.lsClient, deployItem, c.info)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !ShouldReconcile(deployItem) {
		c.log.V(5).Info("aborting reconcile", "phase", deployItem.Status.Phase)
		return reconcile.Result{}, nil
	}
	logger.Info("reconcile deploy item")

	errHdl := HandleErrorFunc(logger, c.lsClient, deployItem)

	switch lsv1alpha1.Operation(lsv1alpha1helper.GetOperation(deployItem.ObjectMeta)) {
	case lsv1alpha1.AbortOperation:
		if err := errHdl(ctx, c.deployer.Abort(ctx, deployItem, target)); err != nil {
			return reconcile.Result{}, err
		}
	case lsv1alpha1.ForceReconcileOperation:
		if err := errHdl(ctx, c.deployer.ForceReconcile(ctx, deployItem, target)); err != nil {
			return reconcile.Result{}, err
		}
	default:

		if !deployItem.DeletionTimestamp.IsZero() {
			if err := errHdl(ctx, c.delete(ctx, deployItem, target)); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, nil
		}

		if err := errHdl(ctx, c.reconcile(ctx, deployItem, target)); err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

func (c *controller) reconcile(ctx context.Context, deployItem *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error {
	if !controllerutil.ContainsFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer)
		if err := c.lsClient.Update(ctx, deployItem); err != nil {
			return lsv1alpha1helper.NewWrappedError(err,
				"Reconcile", "AddFinalizer", err.Error())
		}
	}

	return c.deployer.Reconcile(ctx, deployItem, target)
}

func (c *controller) delete(ctx context.Context, deployItem *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error {
	if err := c.deployer.Delete(ctx, deployItem, target); err != nil {
		return err
	}
	if controllerutil.ContainsFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.RemoveFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer)
		if err := c.lsClient.Update(ctx, deployItem); err != nil {
			return lsv1alpha1helper.NewWrappedError(err,
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
