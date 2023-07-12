// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package healthcheck

import (
	"context"
	"fmt"
	"time"

	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"

	v1 "k8s.io/api/apps/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/config"
)

// NewLsHealthCheckController creates a new health check controller that reconciles the health  object in the namespaces.
func NewLsHealthCheckController(initialLogger logging.Logger, lsDeployments *config.LsDeployments,
	cl client.Client, scheme *runtime.Scheme, durationBorder time.Duration) reconcile.Reconciler {
	return &lsHealthCheckController{
		initialLogger:  initialLogger,
		lsDeployments:  lsDeployments,
		client:         cl,
		scheme:         scheme,
		oldStatus:      lsv1alpha1.LsHealthCheckStatusOk,
		durationBorder: durationBorder,
	}
}

type lsHealthCheckController struct {
	initialLogger  logging.Logger
	lsDeployments  *config.LsDeployments
	client         client.Client
	scheme         *runtime.Scheme
	oldStatus      lsv1alpha1.LsHealthCheckStatus
	durationBorder time.Duration
}

// SetDurationBorder is used for testing.
func (c *lsHealthCheckController) SetDurationBorder(durationBorder time.Duration) {
	c.durationBorder = durationBorder
}

func (c *lsHealthCheckController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger, ctx := c.initialLogger.StartReconcileAndAddToContext(ctx, req)

	if req.Namespace != c.lsDeployments.DeploymentsNamespace || req.Name != c.lsDeployments.LsHealthCheckName {
		return reconcile.Result{}, nil
	}

	// we could assume that the object was created during startup
	lsHealthCheck := &lsv1alpha1.LsHealthCheck{}
	if err := c.client.Get(ctx, req.NamespacedName, lsHealthCheck); err != nil {
		logger.Error(err, "lsHealthCheck object could not be accessed")
		return reconcile.Result{}, err
	}

	if time.Since(lsHealthCheck.LastUpdateTime.Time) < c.durationBorder {
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: c.durationBorder - time.Since(lsHealthCheck.LastUpdateTime.Time) + time.Second,
		}, nil
	} else {
		newStatus, description := c.check(ctx)

		if newStatus == lsv1alpha1.LsHealthCheckStatusOk {
			lsHealthCheck.Status = newStatus
			lsHealthCheck.Description = "ok"
		} else if c.oldStatus == lsv1alpha1.LsHealthCheckStatusFailed && newStatus == lsv1alpha1.LsHealthCheckStatusFailed {
			// Only log the error when the precious state was also in failed.
			// This should help to prevent sporadic errors like "API Service is temporarily not available".
			lsHealthCheck.Status = newStatus
			lsHealthCheck.Description = description
		}

		lsHealthCheck.LastUpdateTime = metav1.Now()

		c.oldStatus = newStatus

		if err := c.client.Update(ctx, lsHealthCheck); err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}
}

func (c *lsHealthCheckController) check(ctx context.Context) (lsv1alpha1.LsHealthCheckStatus, string) {
	if c.lsDeployments != nil {
		isOk, description := c.checkDeployment(ctx, c.lsDeployments.DeploymentsNamespace, c.lsDeployments.LsController)
		if !isOk {
			return lsv1alpha1.LsHealthCheckStatusFailed, description
		}

		isOk, description = c.checkDeployment(ctx, c.lsDeployments.DeploymentsNamespace, c.lsDeployments.WebHook)
		if !isOk {
			return lsv1alpha1.LsHealthCheckStatusFailed, description
		}

		if c.lsDeployments.AdditionalDeployments != nil {
			for _, deployer := range c.lsDeployments.AdditionalDeployments.Deployments {
				isOk, description = c.checkDeployment(ctx, c.lsDeployments.DeploymentsNamespace, deployer)
				if !isOk {
					return lsv1alpha1.LsHealthCheckStatusFailed, description
				}
			}
		}
	}

	return lsv1alpha1.LsHealthCheckStatusOk, "ok"
}

func (c *lsHealthCheckController) checkDeployment(ctx context.Context, namespace string, name string) (bool, string) {
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "checkDeployment"})

	key := client.ObjectKey{Namespace: namespace, Name: name}
	deployment := &v1.Deployment{}
	if err := c.client.Get(ctx, key, deployment); err != nil {
		logger.Error(err, "deployment could not be be fetched", lc.KeyResource, client.ObjectKey{Namespace: namespace, Name: name}.String())
		return false, fmt.Sprintf("deployment %s/%s could not be be fetched", namespace, name)
	}

	expectedReplicas := int32(1)
	if deployment.Spec.Replicas != nil {
		expectedReplicas = *deployment.Spec.Replicas
	}

	if deployment.Generation != deployment.Status.ObservedGeneration ||
		deployment.Status.UpdatedReplicas != expectedReplicas ||
		deployment.Status.AvailableReplicas != expectedReplicas {
		message := fmt.Sprintf("not all pods are running or at the latest state for deployment %s/%s ", namespace, name)
		logger.Info(message)
		return false, message
	}

	return true, "ok"
}
