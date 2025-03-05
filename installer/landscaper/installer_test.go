package landscaper

import (
	"context"
	"github.com/gardener/landscaper/apis/config/v1alpha1"
	"github.com/gardener/landscaper/installer/resources"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Landscaper Controller Installer Test Suite")
}

var _ = Describe("Landscaper Controller Installer", func() {

	const id = "test-g23tp"

	newHostCluster := func() (*resources.Cluster, error) {
		return resources.NewCluster(os.Getenv("KUBECONFIG"))
	}

	It("should install the landscaper controllers", func() {
		ctx := context.Background()

		hostCluster, err := newHostCluster()
		Expect(err).ToNot(HaveOccurred())

		kubeconfig, err := os.ReadFile(os.Getenv("KUBECONFIG"))
		Expect(err).ToNot(HaveOccurred())

		values := &Values{
			Instance:       id,
			Version:        "v0.127.0",
			VerbosityLevel: "INFO",
			Configuration:  v1alpha1.LandscaperConfiguration{},
			ServiceAccount: &ServiceAccountValues{Create: true},
			Controller: ControllerValues{
				LandscaperKubeconfig: &KubeconfigValues{
					Kubeconfig: string(kubeconfig),
				},
				Image: ImageValues{
					Repository: "europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper/github.com/gardener/landscaper/images/landscaper-controller",
					Tag:        "v0.127.0",
				},
				ReplicaCount:  nil,
				Resources:     corev1.ResourceRequirements{},
				ResourcesMain: corev1.ResourceRequirements{},
				Metrics:       nil,
			},
			WebhooksServer: WebhooksServerValues{
				DisableWebhooks: nil,
				LandscaperKubeconfig: &KubeconfigValues{
					Kubeconfig: string(kubeconfig),
				},
				Image: ImageValues{
					Repository: "europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper/github.com/gardener/landscaper/images/landscaper-webhooks-server",
					Tag:        "v0.127.0",
				},
				ServicePort: 0,
				Ingress:     nil,
			},
			ImagePullSecrets:   nil,
			PodSecurityContext: nil,
			SecurityContext:    nil,
			NodeSelector:       nil,
			Affinity:           nil,
			Tolerations:        nil,
		}

		err = InstallLandscaper(ctx, hostCluster, values)
		Expect(err).ToNot(HaveOccurred())
	})

	XIt("should uninstall the landscaper controllers", func() {
		ctx := context.Background()

		hostCluster, err := newHostCluster()
		Expect(err).ToNot(HaveOccurred())

		values := &Values{
			Instance: id,
		}

		err = UninstallLandscaper(ctx, hostCluster, values)
		Expect(err).ToNot(HaveOccurred())
	})

})
