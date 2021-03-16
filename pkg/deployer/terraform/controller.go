// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package terraform

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	terraformv1alpha1 "github.com/gardener/landscaper/apis/deployer/terraform/v1alpha1"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/deployer/targetselector"
	"github.com/gardener/landscaper/pkg/utils/kubernetes"
	"github.com/gardener/landscaper/pkg/version"
)

// NewController creates a new deployer that can handle deploy items of type terraform
func NewController(log logr.Logger, lsClient, hostClient client.Client, hostRestConfig *rest.Config, scheme *runtime.Scheme, config *terraformv1alpha1.Configuration) *controller {

	// default config
	terraformv1alpha1.DefaultConfiguration(config, version.Get().String())

	return &controller{
		log:            log,
		lsClient:       lsClient,
		hostClient:     hostClient,
		hostRestConfig: hostRestConfig,
		scheme:         scheme,
		config:         config,
	}
}

type controller struct {
	log logr.Logger
	// lsClient is the kubernetes client that talks to the landscaper cluster.
	lsClient client.Client
	// hostClient is the kubernetes client that talks to the host cluster where the terraform controller is running.
	hostClient     client.Client
	hostRestConfig *rest.Config
	scheme         *runtime.Scheme
	config         *terraformv1alpha1.Configuration
}

// Reconcile handles the reconcile flow for a terraform deploy item.
func (a *controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := a.log.WithValues("resource", req.NamespacedName)
	logger.V(7).Info("reconcile")

	deployItem := &lsv1alpha1.DeployItem{}
	if err := a.lsClient.Get(ctx, req.NamespacedName, deployItem); err != nil {
		if apierrors.IsNotFound(err) {
			a.log.V(5).Info(err.Error())
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
		if err := a.lsClient.Get(ctx, deployItem.Spec.Target.NamespacedName(), target); err != nil {
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

	err := deployerlib.HandleAnnotationsAndGeneration(ctx, logger, a.lsClient, deployItem)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !deployerlib.ShouldReconcile(deployItem) {
		a.log.V(5).Info("aborting reconcile", "phase", deployItem.Status.Phase)
		return reconcile.Result{}, nil
	}

	old := deployItem.DeepCopy()
	err = a.reconcile(ctx, deployItem, target)
	if !reflect.DeepEqual(old.Status, deployItem.Status) {
		if err := a.lsClient.Status().Update(ctx, deployItem); err != nil {
			a.log.Error(err, "unable to update status")
		}
	}
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (a *controller) reconcile(ctx context.Context, deployItem *lsv1alpha1.DeployItem, target *lsv1alpha1.Target) (err error) {
	old := deployItem.DeepCopy()
	// set failed state if the last error lasts for more than 5 minutes.
	defer func() {
		deployItem.Status.LastError = lsv1alpha1helper.TryUpdateError(deployItem.Status.LastError, err)
		deployItem.Status.Phase = lsv1alpha1.ExecutionPhase(lsv1alpha1helper.GetPhaseForLastError(
			lsv1alpha1.ComponentInstallationPhase(deployItem.Status.Phase),
			deployItem.Status.LastError,
			5*time.Minute))
		// do not return a error so that we not reconcile endlessly if the deploy item is already in a failed state
		if deployItem.Status.Phase == lsv1alpha1.ExecutionPhaseFailed {
			err = nil
		}
		if !reflect.DeepEqual(old.Status, deployItem.Status) {
			if err2 := a.lsClient.Status().Update(ctx, deployItem); err2 != nil {
				if !apierrors.IsConflict(err2) { // reduce logging
					a.log.Error(err2, "unable to update status")
				}
				// retry on conflict
				if err != nil {
					err = err2
				}
			}
		}
	}()

	tf, err := New(a.log, a.lsClient, a.hostClient, a.hostRestConfig, a.config, deployItem, target)
	if err != nil {
		return err
	}

	if !kubernetes.HasFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer) {
		controllerutil.AddFinalizer(deployItem, lsv1alpha1.LandscaperFinalizer)
		if err := a.lsClient.Update(ctx, deployItem); err != nil {
			deployItem.Status.LastError = lsv1alpha1helper.UpdatedError(deployItem.Status.LastError,
				"AddFinalizer", "", err.Error())
			return err
		}
		return nil
	}

	if !deployItem.DeletionTimestamp.IsZero() {
		return tf.Reconcile(ctx, OperationDelete)
	}

	return tf.Reconcile(ctx, OperationReconcile)
}
