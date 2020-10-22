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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/pkg/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	artifactsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/artifacts"
	blueprintsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/blueprints"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/pkg/utils/kubernetes"
	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

func NewActuator(log logr.Logger, lsConfig *config.LandscaperConfiguration) (reconcile.Reconciler, error) {
	op := &operation.Operation{}
	_ = op.InjectLogger(log)

	var (
		regConfig   = &lsConfig.Registries
		sharedCache cache.Cache
	)
	if regConfig.Components.OCI != nil && regConfig.Components.OCI.Cache != nil {
		var err error
		sharedCache, err = cache.NewCache(log, cache.WithConfiguration(regConfig.Components.OCI.Cache))
		if err != nil {
			return nil, err
		}
	}
	componentRegistryMgr, err := componentsregistry.New(sharedCache)
	if err != nil {
		return nil, err
	}
	_ = op.InjectComponentsRegistry(componentRegistryMgr)

	log.V(3).Info("setup components registry")

	if regConfig.Artifacts.OCI != nil && regConfig.Artifacts.OCI.Cache != nil {
		var err error
		sharedCache, err = cache.NewCache(log, cache.WithConfiguration(regConfig.Artifacts.OCI.Cache))
		if err != nil {
			return nil, err
		}
	}
	blueprintsRegistryMgr := blueprintsregistry.New(sharedCache)
	_ = op.InjectBlueprintsRegistry(blueprintsRegistryMgr)
	log.V(3).Info("setup blueprints registry")

	artifactsRegistryMgr, err := artifactsregistry.New(sharedCache)
	if err != nil {
		return nil, err
	}
	_ = op.InjectArtifactsRegistry(artifactsRegistryMgr)
	log.V(3).Info("setup artifacts registry")

	return &actuator{
		Interface:             op,
		lsConfig:              lsConfig,
		componentsRegistryMgr: componentRegistryMgr,
		blueprintRegistryMgr:  blueprintsRegistryMgr,
		artifactsRegistryMgr:  artifactsRegistryMgr,
	}, nil
}

// NewTestActuator creates a new actuator that is only meant for testing.
func NewTestActuator(op operation.Interface, configuration *config.LandscaperConfiguration) *actuator {
	artifactsRegistry, _ := artifactsregistry.New(nil)
	a := &actuator{
		Interface:             op,
		lsConfig:              configuration,
		componentsRegistryMgr: &componentsregistry.Manager{},
		blueprintRegistryMgr:  blueprintsregistry.New(nil),
		artifactsRegistryMgr:  artifactsRegistry,
	}
	return a
}

type actuator struct {
	operation.Interface
	lsConfig              *config.LandscaperConfiguration
	blueprintRegistryMgr  blueprintsregistry.Manager
	componentsRegistryMgr *componentsregistry.Manager
	artifactsRegistryMgr  *artifactsregistry.Manager
}

func (a *actuator) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	defer ctx.Done()
	a.Log().Info("reconcile", "resource", req.NamespacedName)

	inst := &lsv1alpha1.Installation{}
	if err := a.Client().Get(ctx, req.NamespacedName, inst); err != nil {
		if apierrors.IsNotFound(err) {
			a.Log().V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if inst.DeletionTimestamp.IsZero() && !kubernetes.HasFinalizer(inst, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(inst, lsv1alpha1.LandscaperFinalizer)
		if err := a.Client().Update(ctx, inst); err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		return reconcile.Result{}, nil
	}

	// remove the reconcile annotation if it exists
	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ReconcileOperation) {
		delete(inst.Annotations, lsv1alpha1.OperationAnnotation)
		if err := a.Client().Update(ctx, inst); err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		if err := a.reconcile(ctx, inst); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
		if err := a.reconcile(ctx, inst); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.AbortOperation) {
		// todo: handle abort..
		a.Log().Info("do abort")
	}

	if lsv1alpha1helper.IsCompletedInstallationPhase(inst.Status.Phase) && inst.Status.ObservedGeneration == inst.Generation {
		return reconcile.Result{}, nil
	}

	if err := a.reconcile(ctx, inst); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (a *actuator) reconcile(ctx context.Context, inst *lsv1alpha1.Installation) error {
	old := inst.DeepCopy()

	defer func() {
		inst.Status.Phase = lsv1alpha1helper.GetPhaseForLastError(
			inst.Status.Phase,
			inst.Status.LastError,
			5*time.Minute,
		)
		if !reflect.DeepEqual(inst.Status, old.Status) {
			if err := a.Client().Status().Update(ctx, inst); err != nil {
				a.Log().Error(err, "unable to update installation")
			}
		}
	}()

	instOp, err := a.initPrerequisites(ctx, inst)
	if err != nil {
		return err
	}
	internalInstallation := instOp.Inst

	if !inst.DeletionTimestamp.IsZero() {
		return EnsureDeletion(ctx, instOp)
	}

	if lsv1alpha1helper.HasOperation(inst.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
		// need to return and not continue with export validation
		return a.forceReconcile(ctx, instOp, internalInstallation)
	}

	return a.Ensure(ctx, instOp, internalInstallation)
}

func (a *actuator) initPrerequisites(ctx context.Context, inst *lsv1alpha1.Installation) (*installations.Operation, error) {
	if err := a.SetupRegistries(ctx, inst.Spec.RegistryPullSecrets); err != nil {
		inst.Status.LastError = lsv1alpha1helper.UpdatedError(inst.Status.LastError,
			"InitPrerequisites", "SetupRegistries", err.Error())
		return nil, err
	}

	// default repository context if not defined
	if inst.Spec.Blueprint.Reference != nil && inst.Spec.Blueprint.Reference.RepositoryContext == nil {
		inst.Spec.Blueprint.Reference.RepositoryContext = a.lsConfig.RepositoryContext
	}

	intBlueprint, err := blueprints.Resolve(ctx, a.Interface, inst.Spec.Blueprint, nil)
	if err != nil {
		inst.Status.LastError = lsv1alpha1helper.UpdatedError(inst.Status.LastError,
			"InitPrerequisites", "ResolveBlueprint", err.Error())
		return nil, err
	}

	internalInstallation, err := installations.New(inst, intBlueprint)
	if err != nil {
		err = fmt.Errorf("unable to create internal representation of installation: %w", err)
		inst.Status.LastError = lsv1alpha1helper.UpdatedError(inst.Status.LastError,
			"InitPrerequisites", "InitInstallation", err.Error())
		return nil, err
	}

	instOp, err := installations.NewInstallationOperationFromOperation(ctx, a.Interface, internalInstallation)
	if err != nil {
		err = fmt.Errorf("unable to create installation operation: %w", err)
		inst.Status.LastError = lsv1alpha1helper.UpdatedError(inst.Status.LastError,
			"InitPrerequisites", "InitInstallationOperation", err.Error())
		return nil, err
	}
	return instOp, nil
}

func (a *actuator) forceReconcile(ctx context.Context, instOp *installations.Operation, inst *installations.Installation) error {
	inst.Info.Status.Phase = lsv1alpha1.ComponentPhasePending
	if err := a.ApplyUpdate(ctx, instOp, inst); err != nil {
		return err
	}

	delete(inst.Info.Annotations, lsv1alpha1.OperationAnnotation)
	if err := a.Client().Update(ctx, inst.Info); err != nil {
		return err
	}

	inst.Info.Status.ObservedGeneration = inst.Info.Generation
	inst.Info.Status.Phase = lsv1alpha1.ComponentPhaseProgressing
	return nil
}
