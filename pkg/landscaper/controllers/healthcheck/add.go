package healthcheck

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/go-logr/logr"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

// AddControllersToManager adds the healthCheck controller to the manager.
func AddControllersToManager(ctx context.Context, logger logging.Logger, hostMgr manager.Manager, lsDeployments *config.LsDeployments) error {
	cl, err := client.New(hostMgr.GetConfig(), client.Options{Scheme: hostMgr.GetScheme()})
	if err != nil {
		return fmt.Errorf("error building client for healthCheck initialization: %w", err)
	}

	key := client.ObjectKey{Namespace: lsDeployments.DeploymentsNamespace, Name: lsDeployments.LsHealthCheckName}

	lsHealthCheckList := &lsv1alpha1.LsHealthCheckList{}
	if err := cl.List(ctx, lsHealthCheckList, client.InNamespace(lsDeployments.DeploymentsNamespace)); err == nil {
		for _, item := range lsHealthCheckList.Items {
			if item.Name != lsDeployments.LsHealthCheckName {
				if err := cl.Delete(ctx, &item); err != nil {
					return err
				}
			}
		}
	}

	lsHealthCheck := &lsv1alpha1.LsHealthCheck{}
	if err := cl.Get(ctx, key, lsHealthCheck); err != nil {
		if apierrors.IsNotFound(err) {
			lsHealthCheck = &lsv1alpha1.LsHealthCheck{
				ObjectMeta:     metav1.ObjectMeta{Name: lsDeployments.LsHealthCheckName, Namespace: lsDeployments.DeploymentsNamespace},
				Status:         lsv1alpha1.LsHealthCheckStatusInit,
				Description:    "ok",
				LastUpdateTime: metav1.Unix(0, 0),
			}

			if err := cl.Create(ctx, lsHealthCheck); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	log := logger.Reconciles("lsHealthCheck", "LsHealthCheck")
	healthCheckController := NewLsHealthCheckController(log, lsDeployments, cl, hostMgr.GetScheme(), 1*time.Minute)

	err = builder.ControllerManagedBy(hostMgr).
		For(&lsv1alpha1.LsHealthCheck{}).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.Logr() }).
		Complete(healthCheckController)
	if err != nil {
		return fmt.Errorf("unable to register health check controller: %w", err)
	}

	return nil
}
