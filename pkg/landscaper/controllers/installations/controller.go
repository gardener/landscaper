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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/pkg/utils"

	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/utils/kubernetes"
)

const (
	cacheIdentifier = "landscaper-installation-Controller"
)

// NewController creates a new Controller that reconciles Installation resources.
func NewController(log logr.Logger,
	kubeClient client.Client,
	scheme *runtime.Scheme,
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

	op := operation.NewOperation(log, kubeClient, scheme)
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

// Controller is the controller that reconciles a installtion resource.
type Controller struct {
	operation.Operation
	LsConfig            *config.LandscaperConfiguration
	SharedCache         cache.Cache
	ComponentOverwriter componentoverwrites.Overwriter
}

func (c *Controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := c.Log().WithValues("installation", req.NamespacedName.String())
	logger.V(5).Info("reconcile", "resource", req.NamespacedName)

	inst := &lsv1alpha1.Installation{}
	if err := c.Client().Get(ctx, req.NamespacedName, inst); err != nil {
		if apierrors.IsNotFound(err) {
			c.Log().V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	// default the installation as it not done by the Controller runtime
	api.LandscaperScheme.Default(inst)
	errHdl := HandleErrorFunc(logger, c.Client(), inst)

	if inst.DeletionTimestamp.IsZero() && !kubernetes.HasFinalizer(inst, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(inst, lsv1alpha1.LandscaperFinalizer)
		if err := c.Client().Update(ctx, inst); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if !inst.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, errHdl(ctx, c.handleDelete(ctx, inst))
	}

	// remove the reconcile annotation if it exists
	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ReconcileOperation) {
		delete(inst.Annotations, lsv1alpha1.OperationAnnotation)
		if err := c.Client().Update(ctx, inst); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, errHdl(ctx, c.reconcile(ctx, inst))
	}

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
		return reconcile.Result{}, errHdl(ctx, c.forceReconcile(ctx, inst))
	}

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.AbortOperation) {
		// todo: handle abort..
		c.Log().Info("do abort")
	}

	if lsv1alpha1helper.IsCompletedInstallationPhase(inst.Status.Phase) && inst.Status.ObservedGeneration == inst.Generation {
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, errHdl(ctx, c.reconcile(ctx, inst))
}

func (c *Controller) initPrerequisites(ctx context.Context, inst *lsv1alpha1.Installation) (*installations.Operation, error) {
	currOp := "InitPrerequisites"
	op := c.Operation.Copy()
	if err := c.SetupRegistries(ctx, op, inst.Spec.RegistryPullSecrets, inst); err != nil {
		return nil, lsv1alpha1helper.NewWrappedError(err,
			currOp, "SetupRegistries", err.Error())
	}

	// default repository context if not defined
	if err := c.HandleComponentReference(inst); err != nil {
		return nil, err
	}

	cdRef := installations.GetReferenceFromComponentDescriptorDefinition(inst.Spec.ComponentDescriptor)

	intBlueprint, err := blueprints.Resolve(ctx, op.ComponentsRegistry(), cdRef, inst.Spec.Blueprint, nil)
	if err != nil {
		return nil, lsv1alpha1helper.NewWrappedError(err,
			currOp, "ResolveBlueprint", err.Error())
	}

	internalInstallation, err := installations.New(inst, intBlueprint)
	if err != nil {
		err = fmt.Errorf("unable to create internal representation of installation: %w", err)
		return nil, lsv1alpha1helper.NewWrappedError(err,
			currOp, "InitInstallation", err.Error())
	}

	instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, internalInstallation, c.LsConfig.RepositoryContext)
	if err != nil {
		err = fmt.Errorf("unable to create installation operation: %w", err)
		return nil, lsv1alpha1helper.NewWrappedError(err,
			currOp, "InitInstallationOperation", err.Error())
	}
	return instOp, nil
}

// HandleComponentReference defaults and optionally replaces the component reference of a installation.
func (c *Controller) HandleComponentReference(inst *lsv1alpha1.Installation) error {
	if inst.Spec.ComponentDescriptor == nil || inst.Spec.ComponentDescriptor.Reference == nil {
		return nil
	}
	// default repository context if not defined
	if inst.Spec.ComponentDescriptor.Reference.RepositoryContext == nil {
		inst.Spec.ComponentDescriptor.Reference.RepositoryContext = c.LsConfig.RepositoryContext
	}

	if c.ComponentOverwriter == nil {
		return nil
	}

	cond := lsv1alpha1helper.GetOrInitCondition(inst.Status.Conditions, lsv1alpha1.ComponentReferenceOverwriteCondition)
	oldRef := inst.Spec.ComponentDescriptor.Reference.DeepCopy()
	overwritten, err := c.ComponentOverwriter.Replace(inst.Spec.ComponentDescriptor.Reference)
	if err != nil {
		return lsv1alpha1helper.NewWrappedError(err,
			"HandleComponentReference", "OverwriteComponentReference", err.Error())
	}
	if overwritten {
		newRef := inst.Spec.ComponentDescriptor.Reference
		cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue,
			"FoundOverwrite",
			componentoverwrites.ReferenceDiff(oldRef, newRef))
	} else {
		cond = lsv1alpha1helper.UpdatedCondition(cond,
			lsv1alpha1.ConditionFalse,
			"No overwrite defined",
			"component reference has not been overwritten")
	}
	inst.Status.Conditions = lsv1alpha1helper.MergeConditions(inst.Status.Conditions, cond)
	return nil
}

// HandleErrorFunc returns a error handler func for deployers.
// The functions automatically sets the phase for long running errors and updates the status accordingly.
func HandleErrorFunc(log logr.Logger, client client.Client, inst *lsv1alpha1.Installation) func(ctx context.Context, err error) error {
	old := inst.DeepCopy()
	return func(ctx context.Context, err error) error {
		inst.Status.LastError = lsv1alpha1helper.TryUpdateError(inst.Status.LastError, err)
		inst.Status.Phase = lsv1alpha1helper.GetPhaseForLastError(
			inst.Status.Phase,
			inst.Status.LastError,
			5*time.Minute)
		if !reflect.DeepEqual(old.Status, inst.Status) {
			if err2 := client.Status().Update(ctx, inst); err2 != nil {
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
