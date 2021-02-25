// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

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
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/targetselector"
	"github.com/gardener/landscaper/pkg/utils/kubernetes"

	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
)

const (
	cacheIdentifier = "helm-deployer-controller"
)

func NewActuator(log logr.Logger, config *helmv1alpha1.Configuration) (reconcile.Reconciler, error) {

	componentRegistryMgr, err := componentsregistry.SetupManagerFromConfig(log, config.OCI, cacheIdentifier)
	if err != nil {
		return nil, err
	}

	return &actuator{
		log:                   log,
		config:                config,
		componentsRegistryMgr: componentRegistryMgr,
	}, nil
}

type actuator struct {
	log                   logr.Logger
	c                     client.Client
	scheme                *runtime.Scheme
	config                *helmv1alpha1.Configuration
	componentsRegistryMgr *componentsregistry.Manager
}

var _ inject.Client = &actuator{}

var _ inject.Scheme = &actuator{}

// InjectClients injects the current kubernetes registryClient into the actuator
func (a *actuator) InjectClient(c client.Client) error {
	a.c = c
	return nil
}

// InjectScheme injects the current scheme into the actuator
func (a *actuator) InjectScheme(scheme *runtime.Scheme) error {
	a.scheme = scheme
	return nil
}

func (a *actuator) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := a.log.WithValues("resource", req.NamespacedName)
	logger.V(7).Info("reconcile deploy item")

	deployItem := &lsv1alpha1.DeployItem{}
	if err := a.c.Get(ctx, req.NamespacedName, deployItem); err != nil {
		if apierrors.IsNotFound(err) {
			a.log.V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if deployItem.Spec.Type != Type {
		logger.V(7).Info("DeployItem is of wrong type", "type", deployItem.Spec.Type)
		return reconcile.Result{}, nil
	}

	var target *lsv1alpha1.Target
	if deployItem.Spec.Target != nil {
		target = &lsv1alpha1.Target{}
		if err := a.c.Get(ctx, deployItem.Spec.Target.NamespacedName(), target); err != nil {
			return reconcile.Result{}, fmt.Errorf("unable to get target for deploy item: %w", err)
		}
		if len(a.config.TargetSelector) != 0 {
			matched, err := targetselector.Match(target, a.config.TargetSelector)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("unable to match target selector: %w", err)
			}
			if !matched {
				logger.V(5).Info("The deploy item's target does not match the given target selector")
				return reconcile.Result{}, nil
			}
		}
	}

	if deployItem.Status.ObservedGeneration == deployItem.Generation && !lsv1alpha1helper.HasOperation(deployItem.ObjectMeta, lsv1alpha1.ReconcileOperation) {
		logger.V(5).Info("Version already reconciled")
		return reconcile.Result{}, nil
	}

	logger.Info("Reconcile helm deploy item")
	old := deployItem.DeepCopy()
	err := a.reconcile(ctx, deployItem, target)
	if !reflect.DeepEqual(old.Status, deployItem.Status) {
		if err := a.c.Status().Update(ctx, deployItem); err != nil {
			a.log.Error(err, "unable to update status")
		}
	}
	if err != nil {
		return reconcile.Result{}, err
	}

	if lsv1alpha1helper.HasOperation(deployItem.ObjectMeta, lsv1alpha1.ReconcileOperation) {
		delete(deployItem.Annotations, lsv1alpha1.OperationAnnotation)
		if err := a.c.Update(ctx, deployItem); err != nil {
			deployItem.Status.LastError = lsv1alpha1helper.UpdatedError(deployItem.Status.LastError,
				"Reconcile", "RemoveReconcileAnnotation", err.Error())
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (a *actuator) reconcile(ctx context.Context, deployItem *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) (err error) {
	if len(deployItem.Status.Phase) == 0 {
		deployItem.Status.Phase = lsv1alpha1.ExecutionPhaseInit
	}
	// set failed state if the last error lasts for more than 5 minutes
	defer func() {
		// set the error if the err is a landscaper error
		if lsErr, ok := lsv1alpha1helper.IsError(err); ok {
			deployItem.Status.LastError = lsErr.UpdatedError(deployItem.Status.LastError)
		}
		deployItem.Status.Phase = lsv1alpha1.ExecutionPhase(lsv1alpha1helper.GetPhaseForLastError(
			lsv1alpha1.ComponentInstallationPhase(deployItem.Status.Phase),
			deployItem.Status.LastError,
			5*time.Minute))
	}()

	helm, err := New(a.log, a.config, a.c, deployItem, target, a.componentsRegistryMgr)
	if err != nil {
		deployItem.Status.LastError = lsv1alpha1helper.UpdatedError(deployItem.Status.LastError,
			"InitHelmOperation", "", err.Error())
		return err
	}

	if !kubernetes.HasFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer)
		if err := a.c.Update(ctx, deployItem); err != nil {
			deployItem.Status.LastError = lsv1alpha1helper.UpdatedError(deployItem.Status.LastError,
				"AddFinalizer", "", err.Error())
			return err
		}
		return nil
	}

	if !deployItem.DeletionTimestamp.IsZero() {
		return helm.DeleteFiles(ctx)
	}

	files, values, err := helm.Template(ctx)
	if err != nil {
		deployItem.Status.LastError = lsv1alpha1helper.UpdatedError(deployItem.Status.LastError,
			"TemplateChart", "", err.Error())
		return err
	}

	exports, err := helm.constructExportsFromValues(values)
	if err != nil {
		deployItem.Status.LastError = lsv1alpha1helper.UpdatedError(deployItem.Status.LastError,
			"ConstructExportFromValues", "", err.Error())
		return err
	}
	return helm.ApplyFiles(ctx, files, exports)
}
