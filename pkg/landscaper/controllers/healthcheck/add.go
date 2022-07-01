package healthcheck

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// AddControllersToManager adds all deployer registration related deployers to the manager.
func AddControllersToManager(ctx context.Context, log logr.Logger, hostMgr manager.Manager,
	agentConfig *config.AgentConfiguration, lsDeployments *config.LsDeployments, enabledDeployers []string) error {
	lsHealthCheck := &lsv1alpha1.LsHealthCheck{}
	cl := hostMgr.GetClient()
	key := client.ObjectKey{Namespace: agentConfig.Namespace, Name: agentConfig.Name}

	if err := cl.Get(ctx, key, lsHealthCheck); err != nil {
		if apierrors.IsNotFound(err) {
			lsHealthCheck = &lsv1alpha1.LsHealthCheck{
				ObjectMeta:     metav1.ObjectMeta{Name: agentConfig.Name, Namespace: agentConfig.Namespace},
				Status:         lsv1alpha1.LsHealthCheckStatusOk,
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

	healthCheckController := NewLsHealthCheckController(log.WithName("LsHealthCheck"), agentConfig, lsDeployments,
		cl, hostMgr.GetScheme(), enabledDeployers)

	err := builder.ControllerManagedBy(hostMgr).
		For(&lsv1alpha1.LsHealthCheck{}).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.WithName("LsHealthCheck") }).
		Complete(healthCheckController)
	if err != nil {
		return fmt.Errorf("unable to register health check controller: %w", err)
	}

	return nil
}
