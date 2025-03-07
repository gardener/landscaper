package helmdeployer

import (
	"context"
	"github.com/gardener/landscaper/installer/resources"
	"github.com/gardener/landscaper/installer/shared"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Helm Deployer Installer Test Suite")
}

var _ = Describe("Helm Deployer Installer", func() {

	const instanceID = "test-g23tp"

	newHostCluster := func() (*resources.Cluster, error) {
		return resources.NewCluster(os.Getenv("KUBECONFIG"))
	}

	It("should install the helm deployer", func() {
		ctx := context.Background()

		hostCluster, err := newHostCluster()
		Expect(err).ToNot(HaveOccurred())

		kubeconfig, err := os.ReadFile(os.Getenv("KUBECONFIG"))
		Expect(err).ToNot(HaveOccurred())

		values := &Values{
			Instance:    instanceID,
			Version:     "v0.127.0",
			HostCluster: hostCluster,
			LandscaperClusterKubeconfig: &KubeconfigValues{
				Kubeconfig: string(kubeconfig),
			},
			Image: shared.ImageConfig{
				Repository: "europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper/github.com/gardener/landscaper/helm-deployer/images/helm-deployer-controller",
				Tag:        "v0.127.0",
			},
			ImagePullSecrets:       nil,
			PodSecurityContext:     nil,
			SecurityContext:        nil,
			ServiceAccount:         &ServiceAccountValues{Create: true},
			HostClientSettings:     nil,
			ResourceClientSettings: nil,
			NodeSelector:           nil,
		}

		_, err = InstallHelmDeployer(ctx, values)
		Expect(err).ToNot(HaveOccurred())
	})

	XIt("should uninstall the helm deployer", func() {
		ctx := context.Background()

		hostCluster, err := newHostCluster()
		Expect(err).ToNot(HaveOccurred())

		values := &Values{
			Instance:    instanceID,
			HostCluster: hostCluster,
		}

		err = UninstallHelmDeployer(ctx, values)
		Expect(err).ToNot(HaveOccurred())
	})

})
