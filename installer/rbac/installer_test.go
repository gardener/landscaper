package rbac

import (
	"context"
	"github.com/gardener/landscaper/installer/resources"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Landscaper RBAC Installer Test Suite")
}

var _ = Describe("Landscaper RBAC Installer", func() {

	const instanceID = "test-rr8fq"

	newResourceCluster := func() (*resources.Cluster, error) {
		return resources.NewCluster(os.Getenv("RESOURCE_CLUSTER_KUBECONFIG"))
	}

	It("should install the landscaper rbac resources", func() {
		ctx := context.Background()

		resourceCluster, err := newResourceCluster()
		Expect(err).ToNot(HaveOccurred())

		values := &Values{
			Instance:        instanceID,
			Version:         "v0.127.0",
			ResourceCluster: resourceCluster,
			ServiceAccount:  &ServiceAccountValues{Create: true},
		}

		kubeconfigs, err := InstallLandscaperRBACResources(ctx, values)
		Expect(err).ToNot(HaveOccurred())
		Expect(kubeconfigs.ControllerKubeconfig).ToNot(BeNil())
		Expect(kubeconfigs.WebhooksKubeconfig).ToNot(BeNil())
		Expect(kubeconfigs.UserKubeconfig).ToNot(BeNil())
	})

	XIt("should uninstall the landscaper rbac resources", func() {
		ctx := context.Background()

		resourceCluster, err := newResourceCluster()
		Expect(err).ToNot(HaveOccurred())

		values := &Values{
			Instance:        instanceID,
			ResourceCluster: resourceCluster,
		}

		err = UninstallLandscaperRBACResources(ctx, values)
		Expect(err).ToNot(HaveOccurred())
	})

})
