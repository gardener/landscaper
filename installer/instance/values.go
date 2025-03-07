package instance

import (
	"fmt"
	"github.com/gardener/landscaper/apis/config/v1alpha1"
	"github.com/gardener/landscaper/installer/helmdeployer"
	"github.com/gardener/landscaper/installer/landscaper"
	"github.com/gardener/landscaper/installer/manifestdeployer"
	"github.com/gardener/landscaper/installer/rbac"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"os"
)

// rbacValues determines the import values for the installation of the rbac resources
func rbacValues(c *Configuration) *rbac.Values {
	return &rbac.Values{
		Instance:        c.Instance,
		Version:         c.Version,
		ResourceCluster: c.ResourceCluster,
		ServiceAccount:  &rbac.ServiceAccountValues{Create: true},
	}
}

// manifestDeployerValues determines the import values for the installation of the manifest deployer
func manifestDeployerValues(c *Configuration, kubeconfigs *rbac.Kubeconfigs) *manifestdeployer.Values {
	v := &manifestdeployer.Values{
		Instance:       c.Instance,
		Version:        c.Version,
		HostCluster:    c.HostCluster,
		Image:          c.ManifestDeployer.Image,
		ServiceAccount: &manifestdeployer.ServiceAccountValues{Create: true},
		HPA:            c.ManifestDeployer.HPA,
	}

	if kubeconfigs != nil {
		v.LandscaperClusterKubeconfig = &manifestdeployer.KubeconfigValues{
			Kubeconfig: string(kubeconfigs.ControllerKubeconfig),
		}
	}

	return v

}

// helmDeployerValues determines the import values for the installation of the helm deployer
func helmDeployerValues(c *Configuration, kubeconfigs *rbac.Kubeconfigs) *helmdeployer.Values {
	v := &helmdeployer.Values{
		Instance:       c.Instance,
		Version:        c.Version,
		HostCluster:    c.HostCluster,
		Image:          c.HelmDeployer.Image,
		ServiceAccount: &helmdeployer.ServiceAccountValues{Create: true},
		HPA:            c.HelmDeployer.HPA,
	}

	if kubeconfigs != nil {
		v.LandscaperClusterKubeconfig = &helmdeployer.KubeconfigValues{
			Kubeconfig: string(kubeconfigs.ControllerKubeconfig),
		}
	}

	return v
}

// landscaperValues determines the import values for the installation of the landscaper controllers and webhooks server
func landscaperValues(c *Configuration, kubeconfigs *rbac.Kubeconfigs, manifestExports *manifestdeployer.Exports, helmExports *helmdeployer.Exports) *landscaper.Values {
	v := &landscaper.Values{
		Instance:       c.Instance,
		Version:        c.Version,
		HostCluster:    c.HostCluster,
		VerbosityLevel: "INFO",
		Configuration:  v1alpha1.LandscaperConfiguration{},
		ServiceAccount: &landscaper.ServiceAccountValues{Create: true},
		Controller: landscaper.ControllerValues{
			LandscaperKubeconfig: &landscaper.KubeconfigValues{
				Kubeconfig: string(kubeconfigs.ControllerKubeconfig),
			},
			Image:         c.Landscaper.Controller.Image,
			ReplicaCount:  nil,
			Resources:     corev1.ResourceRequirements{},
			ResourcesMain: corev1.ResourceRequirements{},
			Metrics:       nil,
			HPAMain:       c.Landscaper.Controller.HPAMain,
		},
		WebhooksServer: landscaper.WebhooksServerValues{
			DisableWebhooks: nil,
			LandscaperKubeconfig: &landscaper.KubeconfigValues{
				Kubeconfig: string(kubeconfigs.WebhooksKubeconfig),
			},
			Image:       c.Landscaper.WebhooksServer.Image,
			ServicePort: 9443,
			Ingress: &landscaper.IngressValues{
				Host:      fmt.Sprintf("ls-system-%s.%s", c.Instance, os.Getenv("HOST_CLUSTER_DOMAIN")),
				DNSClass:  "garden",
				ClassName: ptr.To("nginx"),
			},
			HPA: c.Landscaper.WebhooksServer.HPA,
		},
	}

	// Deployments to be considered by the health checks
	deployments := []string{}
	if manifestExports != nil {
		deployments = append(deployments, manifestExports.DeploymentName)
	}
	if helmExports != nil {
		deployments = append(deployments, helmExports.DeploymentName)
	}
	v.Controller.HealthChecks = &v1alpha1.AdditionalDeployments{
		Deployments: deployments,
	}

	return v
}
