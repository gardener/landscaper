// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/deployer/lib/targetselector"
	"github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// NewController creates a new deploy item controller that reconciles deploy items of type kubernetes manifest.
func NewController(log logr.Logger, kubeClient client.Client, scheme *runtime.Scheme, config *manifestv1alpha2.Configuration) (reconcile.Reconciler, error) {
	return &controller{
		log:    log,
		client: kubeClient,
		scheme: scheme,
		config: config,
	}, nil
}

type controller struct {
	log    logr.Logger
	client client.Client
	scheme *runtime.Scheme
	config *manifestv1alpha2.Configuration
}

func (a *controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := a.log.WithValues("resource", req.NamespacedName)
	logger.V(7).Info("reconcile")

	deployItem := &lsv1alpha1.DeployItem{}
	if err := a.client.Get(ctx, req.NamespacedName, deployItem); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if deployItem.Spec.Type != Type {
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
				logger.V(5).Info("the deploy item's target has not matched the given target selector",
					"deployItem", deployItem.Name, "target", target.Name)
				return reconcile.Result{}, nil
			}
		}
	}

	logger.Info("reconcile manifest deploy item")

	err := deployerlib.HandleAnnotationsAndGeneration(ctx, logger, a.client, deployItem)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !deployerlib.ShouldReconcile(deployItem) {
		a.log.V(5).Info("aborting reconcile", "phase", deployItem.Status.Phase)
		return reconcile.Result{}, nil
	}

	errHdl := deployerlib.HandleErrorFunc(logger, a.client, deployItem)
	if err := errHdl(ctx, a.reconcile(ctx, deployItem, target)); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (a *controller) reconcile(ctx context.Context, deployItem *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) (err error) {
	logger := a.log.WithValues("resource", types.NamespacedName{Name: deployItem.Name, Namespace: deployItem.Namespace}.String())
	manifest, err := New(logger, a.client, deployItem, target)
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
		return manifest.Delete(ctx)
	}

	return manifest.Reconcile(ctx)
}
