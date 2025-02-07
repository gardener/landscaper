package instance

import (
	"context"
	"github.com/gardener/landscaper/apis/config/v1alpha1"
	"github.com/gardener/landscaper/installer/helmdeployer"
	"github.com/gardener/landscaper/installer/landscaper"
	"github.com/gardener/landscaper/installer/manifestdeployer"
	"github.com/gardener/landscaper/installer/rbac"
	"github.com/gardener/landscaper/installer/resources"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Landscaper Instance Installer Test Suite")
}

var _ = Describe("Landscaper Instance Installer", func() {

	const id = "test2501"

	newHostCluster := func() (*resources.Cluster, error) {
		return resources.NewCluster(os.Getenv("HOST_CLUSTER_KUBECONFIG"))
	}

	newResourceCluster := func() (*resources.Cluster, error) {
		return resources.NewCluster(os.Getenv("RESOURCE_CLUSTER_KUBECONFIG"))
	}

	It("should install the landscaper instance", func() {
		ctx := context.Background()

		hostCluster, err := newHostCluster()
		Expect(err).ToNot(HaveOccurred())

		resourceCluster, err := newResourceCluster()
		Expect(err).ToNot(HaveOccurred())

		values := &Values{
			Instance: id,
			RBACValues: &rbac.Values{
				Instance:       id,
				Version:        "v0.127.0",
				ServiceAccount: &rbac.ServiceAccountValues{Create: true},
			},
			LandscaperValues: &landscaper.Values{
				Instance:       id,
				Version:        "v0.127.0",
				VerbosityLevel: "INFO",
				Configuration:  v1alpha1.LandscaperConfiguration{},
				ServiceAccount: &landscaper.ServiceAccountValues{Create: true},
				Controller: landscaper.ControllerValues{
					LandscaperKubeconfig: &landscaper.KubeconfigValues{
						Kubeconfig: string(resourceCluster.Kubeconfig()),
					},
					Image: landscaper.ImageValues{
						Repository: "europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper/github.com/gardener/landscaper/images/landscaper-controller",
						Tag:        "v0.127.0",
					},
					ReplicaCount:  nil,
					Resources:     corev1.ResourceRequirements{},
					ResourcesMain: corev1.ResourceRequirements{},
					Metrics:       nil,
				},
				WebhooksServer: landscaper.WebhooksServerValues{
					DisableWebhooks: nil,
					LandscaperKubeconfig: &landscaper.KubeconfigValues{
						Kubeconfig: string(resourceCluster.Kubeconfig()),
					},
					Image: landscaper.ImageValues{
						Repository: "europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper/github.com/gardener/landscaper/images/landscaper-webhooks-server",
						Tag:        "v0.127.0",
					},
					ServicePort:           0,
					CertificatesNamespace: "",
					Ingress:               nil,
				},
				ImagePullSecrets:   nil,
				PodSecurityContext: nil,
				SecurityContext:    nil,
				NodeSelector:       nil,
				Affinity:           nil,
				Tolerations:        nil,
			},
			ManifestDeployerValues: &manifestdeployer.Values{
				Instance: id,
				Version:  "v0.127.0",
				LandscaperClusterKubeconfig: &manifestdeployer.KubeconfigValues{
					Kubeconfig: string(resourceCluster.Kubeconfig()),
				},
				Image: manifestdeployer.ImageValues{
					Repository: "europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper/github.com/gardener/landscaper/manifest-deployer/images/manifest-deployer-controller",
					Tag:        "v0.127.0",
				},
				ImagePullSecrets:       nil,
				PodSecurityContext:     nil,
				SecurityContext:        nil,
				ServiceAccount:         &manifestdeployer.ServiceAccountValues{Create: true},
				HostClientSettings:     nil,
				ResourceClientSettings: nil,
				NodeSelector:           nil,
			},
			HelmDeployerValues: &helmdeployer.Values{
				Instance: id,
				Version:  "v0.127.0",
				LandscaperClusterKubeconfig: &helmdeployer.KubeconfigValues{
					Kubeconfig: string(resourceCluster.Kubeconfig()),
				},
				Image: helmdeployer.ImageValues{
					Repository: "europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper/github.com/gardener/landscaper/helm-deployer/images/helm-deployer-controller",
					Tag:        "v0.127.0",
				},
				ImagePullSecrets:       nil,
				PodSecurityContext:     nil,
				SecurityContext:        nil,
				ServiceAccount:         &helmdeployer.ServiceAccountValues{Create: true},
				HostClientSettings:     nil,
				ResourceClientSettings: nil,
				NodeSelector:           nil,
			},
		}

		err = InstallLandscaperInstance(ctx, hostCluster, resourceCluster, values)
		Expect(err).ToNot(HaveOccurred())
	})

	XIt("should uninstall the landscaper instance", func() {
		ctx := context.Background()

		hostCluster, err := newHostCluster()
		Expect(err).ToNot(HaveOccurred())

		resourceCluster, err := newResourceCluster()
		Expect(err).ToNot(HaveOccurred())

		values := &Values{
			Instance:               id,
			RBACValues:             &rbac.Values{Instance: id},
			LandscaperValues:       &landscaper.Values{Instance: id},
			ManifestDeployerValues: &manifestdeployer.Values{Instance: id},
			HelmDeployerValues:     &helmdeployer.Values{Instance: id},
		}

		err = UninstallLandscaperInstance(ctx, hostCluster, resourceCluster, values)
		Expect(err).ToNot(HaveOccurred())
	})

})
