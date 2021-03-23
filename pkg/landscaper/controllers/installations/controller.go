// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/pkg/utils/kubernetes"
)

const (
	cacheIdentifier = "landscaper-installation-controller"
)

// NewController creates a new controller that reconciles Installation resources.
func NewController(log logr.Logger, kubeClient client.Client, scheme *runtime.Scheme, lsConfig *config.LandscaperConfiguration) (reconcile.Reconciler, error) {
	componentRegistryMgr, err := componentsregistry.SetupManagerFromConfig(log, lsConfig.Registry.OCI, cacheIdentifier)
	if err != nil {
		return nil, err
	}
	log.V(3).Info("setup components registry")

	op := operation.NewOperation(log, kubeClient, scheme, componentRegistryMgr)
	return &controller{
		Interface:             op,
		lsConfig:              lsConfig,
		componentsRegistryMgr: componentRegistryMgr,
	}, nil
}

// NewTestActuator creates a new controller that is only meant for testing.
func NewTestActuator(op operation.Interface, configuration *config.LandscaperConfiguration) *controller {
	a := &controller{
		Interface:             op,
		lsConfig:              configuration,
		componentsRegistryMgr: &componentsregistry.Manager{},
	}
	resolver := op.ComponentsRegistry().(componentsregistry.TypedRegistry)
	err := a.componentsRegistryMgr.Set(resolver)
	if err != nil {
		return nil
	}
	err = operation.InjectComponentsRegistryInto(op, a.componentsRegistryMgr)
	if err != nil {
		return nil
	}

	return a
}

type controller struct {
	operation.Interface
	lsConfig              *config.LandscaperConfiguration
	componentsRegistryMgr *componentsregistry.Manager
}

func (c *controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
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
	// default the installation as it not done by the controller runtime
	lsv1alpha1.SetDefaults_Installation(inst)
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

func (c *controller) initPrerequisites(ctx context.Context, inst *lsv1alpha1.Installation) (*installations.Operation, error) {
	currOp := "InitPrerequisites"
	if err := c.SetupRegistries(ctx, inst.Spec.RegistryPullSecrets, inst); err != nil {
		return nil, lsv1alpha1helper.NewWrappedError(err,
			currOp, "SetupRegistries", err.Error())
	}

	// default repository context if not defined
	if inst.Spec.ComponentDescriptor != nil && inst.Spec.ComponentDescriptor.Reference != nil && inst.Spec.ComponentDescriptor.Reference.RepositoryContext == nil {
		inst.Spec.ComponentDescriptor.Reference.RepositoryContext = c.lsConfig.RepositoryContext
	}

	cdRef := installations.GeReferenceFromComponentDescriptorDefinition(inst.Spec.ComponentDescriptor)

	intBlueprint, err := blueprints.Resolve(ctx, c.Interface.ComponentsRegistry(), cdRef, inst.Spec.Blueprint, nil)
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

	instOp, err := installations.NewInstallationOperationFromOperation(ctx, c.Interface, internalInstallation)
	if err != nil {
		err = fmt.Errorf("unable to create installation operation: %w", err)
		return nil, lsv1alpha1helper.NewWrappedError(err,
			currOp, "InitInstallationOperation", err.Error())
	}
	return instOp, nil
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
