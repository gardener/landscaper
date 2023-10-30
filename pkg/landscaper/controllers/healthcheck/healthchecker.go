// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package healthcheck

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
)

const healthCheckInterval = time.Minute

type HealthChecker struct {
	lsDeployments *config.LsDeployments
	hostClient    client.Client
	oldStatus     lsv1alpha1.LsHealthCheckStatus
}

func NewHealthChecker(lsDeployments *config.LsDeployments, hostClient client.Client) *HealthChecker {
	return &HealthChecker{
		lsDeployments: lsDeployments,
		hostClient:    hostClient,
		oldStatus:     lsv1alpha1.LsHealthCheckStatusOk,
	}
}

// StartPeriodicalHealthCheck first initializes and then starts the periodical health check. A landscaper instance
// has a unique LsHealthCheck resource on its host cluster (namespace: lsDeployments.DeploymentsNamespace,
// name: lsDeployments.LsHealthCheckName). The initialization creates this resource if it does not yet exist, and
// deletes all other LsHealthCheck resources in the same namespace. If the initialization fails, the function returns
// an error. If it succeeds, it starts the periodical health check. In this case the function does not return until
// the context is cancelled.
// Each execution of the healthcheck checks the number of replicas of the Deployments listed in lsDeployments, and
// updates the status of the LsHealthCheck resource. Note however, that a failure is only written to the LsHealthCheck
// resource if it occurs twice in succession.
func (c *HealthChecker) StartPeriodicalHealthCheck(ctx context.Context, logger logging.Logger) error {
	log := logger.WithName("lsHealthCheck").WithValues(lc.KeyReconciledResource, c.healthCheckKey().String())
	ctx = logging.NewContext(ctx, log)

	if err := c.initializeHealthCheck(ctx); err != nil {
		log.Error(err, "error during healthcheck initialization")
		return fmt.Errorf("error during healthcheck initialization: %w", err)
	}

	log.Info("starting periodical healthcheck")
	wait.UntilWithContext(ctx, c.ExecuteHealthCheck, healthCheckInterval)

	return nil
}

func (c *HealthChecker) healthCheckKey() client.ObjectKey {
	return client.ObjectKey{Namespace: c.lsDeployments.DeploymentsNamespace, Name: c.lsDeployments.LsHealthCheckName}
}

// initializeHealthCheck ensures that there is exactly one LsHealthCheck resource for the present landscaper.
func (c *HealthChecker) initializeHealthCheck(ctx context.Context) error {
	log, ctx := logging.FromContextOrNew(ctx, nil, lc.KeyMethod, "initializeHealthCheck")
	log.Info("initializing healthcheck")

	lsHealthCheckList := &lsv1alpha1.LsHealthCheckList{}
	if err := c.hostClient.List(ctx, lsHealthCheckList, client.InNamespace(c.lsDeployments.DeploymentsNamespace)); err == nil {
		for _, item := range lsHealthCheckList.Items {
			if item.Name != c.lsDeployments.LsHealthCheckName {
				if err := c.hostClient.Delete(ctx, &item); err != nil {
					return fmt.Errorf("error deleting lsHealthCheck resource %s: %w", client.ObjectKeyFromObject(&item).String(), err)
				}
			}
		}
	}

	lsHealthCheck := &lsv1alpha1.LsHealthCheck{}
	if err := c.hostClient.Get(ctx, c.healthCheckKey(), lsHealthCheck); err != nil {
		if apierrors.IsNotFound(err) {
			lsHealthCheck = &lsv1alpha1.LsHealthCheck{
				ObjectMeta:     metav1.ObjectMeta{Name: c.lsDeployments.LsHealthCheckName, Namespace: c.lsDeployments.DeploymentsNamespace},
				Status:         lsv1alpha1.LsHealthCheckStatusInit,
				Description:    "ok",
				LastUpdateTime: metav1.Unix(0, 0),
			}

			if err := c.hostClient.Create(ctx, lsHealthCheck); err != nil {
				return fmt.Errorf("error creating lsHealthCheck resource: %w", err)
			}
		} else {
			return fmt.Errorf("error reading lsHealthCheck resource: %w", err)
		}
	}

	return nil
}

func (c *HealthChecker) ExecuteHealthCheck(ctx context.Context) {
	log, ctx := logging.FromContextOrNew(ctx, nil)

	// The healthcheck object was created during initialization.
	lsHealthCheck := &lsv1alpha1.LsHealthCheck{}
	if err := c.hostClient.Get(ctx, c.healthCheckKey(), lsHealthCheck); err != nil {
		log.Error(err, "lsHealthCheck object could not be accessed")
		return
	}

	newStatus, description := c.check(ctx)

	if newStatus == lsv1alpha1.LsHealthCheckStatusOk {
		lsHealthCheck.Status = newStatus
		lsHealthCheck.Description = "ok"
	} else if c.oldStatus == lsv1alpha1.LsHealthCheckStatusFailed && newStatus == lsv1alpha1.LsHealthCheckStatusFailed {
		// Only log the error when the previous state was also in failed.
		// This should help to prevent sporadic errors like "API Service is temporarily not available".
		lsHealthCheck.Status = newStatus
		lsHealthCheck.Description = description
	}

	lsHealthCheck.LastUpdateTime = metav1.Now()

	c.oldStatus = newStatus

	if err := c.hostClient.Update(ctx, lsHealthCheck); err != nil {
		log.Error(err, "error updating lsHealthCheck object")
	}
}

func (c *HealthChecker) check(ctx context.Context) (lsv1alpha1.LsHealthCheckStatus, string) {
	isOk, description := c.checkDeployment(ctx, c.lsDeployments.DeploymentsNamespace, c.lsDeployments.LsController)
	if !isOk {
		return lsv1alpha1.LsHealthCheckStatusFailed, description
	}

	isOk, description = c.checkDeployment(ctx, c.lsDeployments.DeploymentsNamespace, c.lsDeployments.LsMainController)
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

	return lsv1alpha1.LsHealthCheckStatusOk, "ok"
}

func (c *HealthChecker) checkDeployment(ctx context.Context, namespace string, name string) (bool, string) {
	log, ctx := logging.FromContextOrNew(ctx, nil, lc.KeyMethod, "checkDeployment")

	deploymentKey := client.ObjectKey{Namespace: namespace, Name: name}
	deployment := &v1.Deployment{}
	if err := c.hostClient.Get(ctx, deploymentKey, deployment); err != nil {
		description := fmt.Sprintf("deployment %s could not be be fetched", deploymentKey.String())
		log.Error(err, description)
		return false, description
	}

	expectedReplicas := int32(1)
	if deployment.Spec.Replicas != nil {
		expectedReplicas = *deployment.Spec.Replicas
	}

	if deployment.Generation != deployment.Status.ObservedGeneration ||
		deployment.Status.UpdatedReplicas != expectedReplicas ||
		deployment.Status.AvailableReplicas != expectedReplicas {
		description := fmt.Sprintf("not all pods are running or at the latest state for deployment %s ", deploymentKey.String())
		log.Info(description)
		return false, description
	}

	return true, "ok"
}
