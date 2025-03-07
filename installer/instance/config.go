package instance

import (
	"github.com/gardener/landscaper/installer/resources"
	"github.com/gardener/landscaper/installer/shared"
)

const (
	helm     = "helm"
	manifest = "manifest"
)

// newConfiguration creates a  Configuration which is partially filled, namely with the instance independent values.
func newConfiguration() *Configuration {
	return &Configuration{
		Version: "v0.127.0",
		Landscaper: LandscaperConfig{
			Controller: ControllerConfig{
				Image: shared.ImageConfig{
					Repository: "europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper/github.com/gardener/landscaper/images/landscaper-controller",
					Tag:        "v0.127.0",
				},
			},
			WebhooksServer: WebhooksServerConfig{
				Image: shared.ImageConfig{
					Repository: "europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper/github.com/gardener/landscaper/images/landscaper-webhooks-server",
					Tag:        "v0.127.0",
				},
			},
		},
		ManifestDeployer: &ManifestDeployerConfig{
			Image: shared.ImageConfig{
				Repository: "europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper/github.com/gardener/landscaper/manifest-deployer/images/manifest-deployer-controller",
				Tag:        "v0.127.0",
			},
		},
		HelmDeployer: &HelmDeployerConfig{
			Image: shared.ImageConfig{
				Repository: "europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper/github.com/gardener/landscaper/helm-deployer/images/helm-deployer-controller",
				Tag:        "v0.127.0",
			},
		},
	}
}

type Configuration struct {
	Instance shared.Instance
	Version  string

	HostCluster     *resources.Cluster
	ResourceCluster *resources.Cluster

	// Deployers is the list of deployers that are getting installed alongside with this Instance.
	// Supported deployers are: "helm", "manifest".
	Deployers []string

	Landscaper LandscaperConfig

	ManifestDeployer *ManifestDeployerConfig

	HelmDeployer *HelmDeployerConfig
}

type LandscaperConfig struct {
	Controller     ControllerConfig
	WebhooksServer WebhooksServerConfig
}

type ControllerConfig struct {
	// Image specifies the image of the landscaper controller.
	Image shared.ImageConfig
}

type WebhooksServerConfig struct {
	// Image specifies the image of the webhooks server.
	Image shared.ImageConfig
}

type ManifestDeployerConfig struct {
	// Image specifies the image of the manifest deployer controller.
	Image shared.ImageConfig
}

type HelmDeployerConfig struct {
	// Image specifies the image of the helm deployer controller.
	Image shared.ImageConfig
}
