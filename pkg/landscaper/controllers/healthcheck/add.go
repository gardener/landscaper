package healthcheck

import (
	"context"
	"fmt"

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

// AddControllersToManager adds all deployer registration related deployers to the manager.
func AddControllersToManager(ctx context.Context, logger logging.Logger, hostMgr manager.Manager,
	agentConfig *config.AgentConfiguration, lsDeployments *config.LsDeployments, enabledDeployers []string) error {
	lsHealthCheck := &lsv1alpha1.LsHealthCheck{}
	cl := hostMgr.GetClient()
	key := client.ObjectKey{Namespace: agentConfig.Namespace, Name: agentConfig.Name}

	if err := cl.Get(ctx, key, lsHealthCheck); err != nil {
		if apierrors.IsNotFound(err) {
			lsHealthCheck = &lsv1alpha1.LsHealthCheck{
				ObjectMeta:     metav1.ObjectMeta{Name: agentConfig.Name, Namespace: agentConfig.Namespace},
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
	healthCheckController := NewLsHealthCheckController(log, agentConfig, lsDeployments,
		cl, hostMgr.GetScheme(), enabledDeployers)

	err := builder.ControllerManagedBy(hostMgr).
		For(&lsv1alpha1.LsHealthCheck{}).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.Logr() }).
		Complete(healthCheckController)
	if err != nil {
		return fmt.Errorf("unable to register health check controller: %w", err)
	}

	return nil
}
