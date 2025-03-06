package instance

import (
	"context"
	"fmt"
	"github.com/gardener/landscaper/apis/config/v1alpha1"
	"github.com/gardener/landscaper/installer/helmdeployer"
	"github.com/gardener/landscaper/installer/landscaper"
	"github.com/gardener/landscaper/installer/manifestdeployer"
	"github.com/gardener/landscaper/installer/rbac"
	"github.com/gardener/landscaper/installer/resources"
)

func InstallLandscaperInstance(ctx context.Context, hostCluster, resourceCluster *resources.Cluster, values *Values) error {

	kubeconfigs, err := rbac.InstallLandscaperRBACResources(ctx, resourceCluster, rbacValues(values))
	if err != nil {
		return fmt.Errorf("failed to install landscaper rbac resources: %v", err)
	}

	manifestExports, err := manifestdeployer.InstallManifestDeployer(ctx, hostCluster, manifestDeployerValues(values, kubeconfigs))
	if err != nil {
		return fmt.Errorf("failed to install manifest deployer: %w", err)
	}

	helmExports, err := helmdeployer.InstallHelmDeployer(ctx, hostCluster, helmDeployerValues(values, kubeconfigs))
	if err != nil {
		return fmt.Errorf("failed to install helm deployer: %w", err)
	}

	err = landscaper.InstallLandscaper(ctx, hostCluster, landscaperValues(values, kubeconfigs, manifestExports, helmExports))
	if err != nil {
		return fmt.Errorf("failed to install landscaper controllers: %w", err)
	}

	return nil
}

func UninstallLandscaperInstance(ctx context.Context, hostCluster, resourceCluster *resources.Cluster, values *Values) error {
	kubeconfigs := &rbac.Kubeconfigs{}

	err := landscaper.UninstallLandscaper(ctx, hostCluster, landscaperValues(values, kubeconfigs, nil, nil))
	if err != nil {
		return fmt.Errorf("failed to uninstall landscaper controllers: %w", err)
	}

	err = helmdeployer.UninstallHelmDeployer(ctx, hostCluster, helmDeployerValues(values, kubeconfigs))
	if err != nil {
		return fmt.Errorf("failed to uninstall helm deployer: %w", err)
	}

	err = manifestdeployer.UninstallManifestDeployer(ctx, hostCluster, manifestDeployerValues(values, kubeconfigs))
	if err != nil {
		return fmt.Errorf("failed to uninstall manifest deployer: %w", err)
	}

	err = rbac.UninstallLandscaperRBACResources(ctx, resourceCluster, rbacValues(values))
	if err != nil {
		return fmt.Errorf("failed to uninstall landscaper rbac resources: %v", err)
	}

	return nil
}

func rbacValues(v *Values) *rbac.Values {
	return v.RBACValues
}

func landscaperValues(v *Values, kubeconfigs *rbac.Kubeconfigs, manifestExports *manifestdeployer.Exports, helmExports *helmdeployer.Exports) *landscaper.Values {
	lv := v.LandscaperValues
	if lv == nil {
		lv = &landscaper.Values{}
	}
	if lv.Controller.LandscaperKubeconfig == nil {
		lv.Controller.LandscaperKubeconfig = &landscaper.KubeconfigValues{}
	}
	lv.Controller.LandscaperKubeconfig.Kubeconfig = string(kubeconfigs.ControllerKubeconfig)

	if lv.WebhooksServer.LandscaperKubeconfig == nil {
		lv.WebhooksServer.LandscaperKubeconfig = &landscaper.KubeconfigValues{}
	}
	lv.WebhooksServer.LandscaperKubeconfig.Kubeconfig = string(kubeconfigs.WebhooksKubeconfig)

	// deployments to be considered by the health checks
	deployments := []string{}
	if manifestExports != nil {
		deployments = append(deployments, manifestExports.DeploymentName)
	}
	if helmExports != nil {
		deployments = append(deployments, helmExports.DeploymentName)
	}
	lv.Controller.HealthChecks = &v1alpha1.AdditionalDeployments{
		Deployments: deployments,
	}

	return lv
}

func manifestDeployerValues(v *Values, kubeconfigs *rbac.Kubeconfigs) *manifestdeployer.Values {
	mv := v.ManifestDeployerValues
	if mv == nil {
		mv = &manifestdeployer.Values{}
	}
	if mv.LandscaperClusterKubeconfig == nil {
		mv.LandscaperClusterKubeconfig = &manifestdeployer.KubeconfigValues{}
	}
	mv.LandscaperClusterKubeconfig.Kubeconfig = string(kubeconfigs.ControllerKubeconfig)

	return mv
}

func helmDeployerValues(v *Values, kubeconfigs *rbac.Kubeconfigs) *helmdeployer.Values {
	hv := v.HelmDeployerValues
	if hv == nil {
		hv = &helmdeployer.Values{}
	}
	if hv.LandscaperClusterKubeconfig == nil {
		hv.LandscaperClusterKubeconfig = &helmdeployer.KubeconfigValues{}
	}
	hv.LandscaperClusterKubeconfig.Kubeconfig = string(kubeconfigs.ControllerKubeconfig)

	return hv
}
