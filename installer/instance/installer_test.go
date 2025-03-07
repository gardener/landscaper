package instance

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
	RunSpecs(t, "Landscaper Instance Installer Test Suite")
}

var _ = Describe("Landscaper Instance Installer", func() {

	const instanceID = "test2501"

	newHostCluster := func() (*resources.Cluster, error) {
		return resources.NewCluster(os.Getenv("HOST_CLUSTER_KUBECONFIG"))
	}

	newResourceCluster := func() (*resources.Cluster, error) {
		return resources.NewCluster(os.Getenv("RESOURCE_CLUSTER_KUBECONFIG"))
	}

	It("should install the landscaper instance", func() {
		var err error
		ctx := context.Background()

		// Create configuration with instance independent values
		config := newConfiguration()

		// Add instance specific values
		config.Instance = instanceID
		config.Deployers = []string{manifest, helm}
		config.HostCluster, err = newHostCluster()
		Expect(err).ToNot(HaveOccurred())
		config.ResourceCluster, err = newResourceCluster()
		Expect(err).ToNot(HaveOccurred())

		// Add optional values
		config.HelmDeployer.HPA = shared.HPAValues{
			MaxReplicas: 3,
		}

		err = InstallLandscaperInstance(ctx, config)
		Expect(err).ToNot(HaveOccurred())
	})

	XIt("should uninstall the landscaper instance", func() {
		var err error
		ctx := context.Background()

		// Create configuration with instance independent values
		config := newConfiguration()

		// Add instance specific values
		config.Instance = instanceID
		config.Deployers = []string{manifest, helm}
		config.HostCluster, err = newHostCluster()
		Expect(err).ToNot(HaveOccurred())
		config.ResourceCluster, err = newResourceCluster()
		Expect(err).ToNot(HaveOccurred())

		err = UninstallLandscaperInstance(ctx, config)
		Expect(err).ToNot(HaveOccurred())
	})

})
