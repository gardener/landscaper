// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"fmt"

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
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/deployer/targetselector"
	"github.com/gardener/landscaper/pkg/utils/kubernetes"

	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
)

const (
	cacheIdentifier = "helm-deployer-controller"
)

// NewController creates a new deploy item controller that reconciles deploy items of type helm.
func NewController(log logr.Logger, kubeClient client.Client, scheme *runtime.Scheme, config *helmv1alpha1.Configuration) (reconcile.Reconciler, error) {

	componentRegistryMgr, err := componentsregistry.SetupManagerFromConfig(log, config.OCI, cacheIdentifier)
	if err != nil {
		return nil, err
	}

	return &controller{
		log:                   log,
		client:                kubeClient,
		scheme:                scheme,
		config:                config,
		componentsRegistryMgr: componentRegistryMgr,
	}, nil
}

type controller struct {
	log                   logr.Logger
	client                client.Client
	scheme                *runtime.Scheme
	config                *helmv1alpha1.Configuration
	componentsRegistryMgr *componentsregistry.Manager
}

var _ inject.Client = &controller{}

var _ inject.Scheme = &controller{}

// InjectClients injects the current kubernetes registryClient into the controller
func (a *controller) InjectClient(c client.Client) error {
	a.client = c
	return nil
}

// InjectScheme injects the current scheme into the controller
func (a *controller) InjectScheme(scheme *runtime.Scheme) error {
	a.scheme = scheme
	return nil
}

func (a *controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := a.log.WithValues("resource", req.NamespacedName)
	logger.V(7).Info("reconcile deploy item")

	deployItem := &lsv1alpha1.DeployItem{}
	if err := a.client.Get(ctx, req.NamespacedName, deployItem); err != nil {
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
		if err := a.client.Get(ctx, deployItem.Spec.Target.NamespacedName(), target); err != nil {
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
	errHdl := deployerlib.HandleErrorFunc(logger, a.client, deployItem)
	if err := errHdl(ctx, a.reconcile(ctx, deployItem, target)); err != nil {
		return reconcile.Result{}, err
	}

	if lsv1alpha1helper.HasOperation(deployItem.ObjectMeta, lsv1alpha1.ReconcileOperation) {
		delete(deployItem.Annotations, lsv1alpha1.OperationAnnotation)
		if err := a.client.Update(ctx, deployItem); err != nil {
			deployItem.Status.LastError = lsv1alpha1helper.UpdatedError(deployItem.Status.LastError,
				"Reconcile", "RemoveReconcileAnnotation", err.Error())
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (a *controller) reconcile(ctx context.Context, deployItem *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) error {
	if len(deployItem.Status.Phase) == 0 {
		deployItem.Status.Phase = lsv1alpha1.ExecutionPhaseInit
	}

	helm, err := New(a.log, a.config, a.client, deployItem, target, a.componentsRegistryMgr)
	if err != nil {
		return err
	}

	if !kubernetes.HasFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer)
		if err := a.client.Update(ctx, deployItem); err != nil {
			return lsv1alpha1helper.NewWrappedError(err,
				"Reconcile", "AddFinalizer", err.Error(), lsv1alpha1.ErrorConfigurationProblem)
		}
		return nil
	}

	if !deployItem.DeletionTimestamp.IsZero() {
		return helm.DeleteFiles(ctx)
	}

	files, values, err := helm.Template(ctx)
	if err != nil {
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
