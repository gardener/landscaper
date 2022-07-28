// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package healthcheck

import (
	"context"
	"fmt"
	"time"

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
func NewLsHealthCheckController(initialLogger logging.Logger, agentConfig *config.AgentConfiguration, lsDeployments *config.LsDeployments,
	cl client.Client, scheme *runtime.Scheme, enabledDeployers []string) reconcile.Reconciler {
	return &lsHealthCheckController{
		initialLogger:    initialLogger,
		agentConfig:      agentConfig,
		lsDeployments:    lsDeployments,
		client:           cl,
		scheme:           scheme,
		enabledDeployers: enabledDeployers,
		oldStatus:        lsv1alpha1.LsHealthCheckStatusOk,
	}
}

type lsHealthCheckController struct {
	initialLogger    logging.Logger
	agentConfig      *config.AgentConfiguration
	lsDeployments    *config.LsDeployments
	client           client.Client
	scheme           *runtime.Scheme
	enabledDeployers []string
	oldStatus        lsv1alpha1.LsHealthCheckStatus
}

func (c *lsHealthCheckController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := c.initialLogger.StartReconcile(req)

	if req.Namespace != c.agentConfig.Namespace || req.Name != c.agentConfig.Name {
		return reconcile.Result{}, nil
	}

	// we could assume that the object was created during startup
	lsHealthCheck := &lsv1alpha1.LsHealthCheck{}
	if err := c.client.Get(ctx, req.NamespacedName, lsHealthCheck); err != nil {
		logger.Error(err, "lsHealthCheck object could not be accessed")
		return reconcile.Result{}, err
	}

	durationBorder := time.Minute * 1

	if time.Since(lsHealthCheck.LastUpdateTime.Time) < durationBorder {
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: durationBorder - time.Since(lsHealthCheck.LastUpdateTime.Time) + time.Second,
		}, nil
	} else {
		newStatus, description := c.check(ctx, logger)

		if newStatus == lsv1alpha1.LsHealthCheckStatusOk {
			lsHealthCheck.Status = newStatus
			lsHealthCheck.Description = "ok"
		} else if c.oldStatus == lsv1alpha1.LsHealthCheckStatusFailed && newStatus == lsv1alpha1.LsHealthCheckStatusFailed {
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

func (c *lsHealthCheckController) check(ctx context.Context, log logging.Logger) (lsv1alpha1.LsHealthCheckStatus, string) {
	if c.lsDeployments != nil {
		isOk, description := c.checkDeployment(ctx, c.agentConfig.Namespace, c.lsDeployments.LsController, log)
		if !isOk {
			return lsv1alpha1.LsHealthCheckStatusFailed, description
		}

		isOk, description = c.checkDeployment(ctx, c.agentConfig.Namespace, c.lsDeployments.WebHook, log)
		if !isOk {
			return lsv1alpha1.LsHealthCheckStatusFailed, description
		}

		for _, deployer := range c.enabledDeployers {
			deploymentName := deployer + "-" + c.agentConfig.Name + "-" + deployer + "-deployer"
			isOk, description = c.checkDeployment(ctx, c.agentConfig.Namespace, deploymentName, log)
			if !isOk {
				return lsv1alpha1.LsHealthCheckStatusFailed, description
			}
		}
	}

	return lsv1alpha1.LsHealthCheckStatusOk, "ok"
}

func (c *lsHealthCheckController) checkDeployment(ctx context.Context, namespace string, name string, log logging.Logger) (bool, string) {
	key := client.ObjectKey{Namespace: namespace, Name: name}
	deployment := &v1.Deployment{}
	if err := c.client.Get(ctx, key, deployment); err != nil {
		message := fmt.Sprintf("deployment %s/%s could not be be fetched", namespace, name)
		log.Error(err, message)
		return false, message
	}

	if deployment.Generation != deployment.Status.ObservedGeneration ||
		deployment.Status.UpdatedReplicas != 1 ||
		deployment.Status.AvailableReplicas != 1 {
		message := fmt.Sprintf("not all pods are running or at the latest state for deployment %s/%s ", namespace, name)
		return false, message
	}

	return true, "ok"
}
