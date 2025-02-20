package helmdeployer

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"testing"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Helm Deployer Installer Test Suite")
}

var _ = Describe("Helm Deployer Installer", func() {

	const id = "test-g23tp"

	newHostClient := func() (client.Client, error) {
		cfg, err := config.GetConfig()
		if err != nil {
			return nil, fmt.Errorf("unable to load kubeconfig for host cluster of helm deployer: %v\n", err)
		}
		hostClient, err := client.New(cfg, client.Options{})
		if err != nil {
			return nil, fmt.Errorf("unable to create kubernetes client for host cluster of helm deployer: %v\n", err)
		}
		return hostClient, nil
	}

	It("should install the helm deployer", func() {
		ctx := context.Background()

		kubeconfig, err := os.ReadFile(os.Getenv("KUBECONFIG"))
		Expect(err).ToNot(HaveOccurred())

		values := &Values{
			Key:     NewKeyFromID(id),
			Version: "v0.127.0",
			LandscaperClusterKubeconfig: &KubeconfigValues{
				Kubeconfig: string(kubeconfig),
			},
			Image: ImageValues{
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

		hostCl, err := newHostClient()
		Expect(err).ToNot(HaveOccurred())

		err = InstallHelmDeployer(ctx, hostCl, values)
		Expect(err).ToNot(HaveOccurred())
	})

	XIt("should uninstall the helm deployer", func() {
		ctx := context.Background()

		values := &Values{
			Key: NewKeyFromID(id),
		}

		hostCl, err := newHostClient()
		Expect(err).ToNot(HaveOccurred())

		err = UninstallHelmDeployer(ctx, hostCl, values)
		Expect(err).ToNot(HaveOccurred())
	})

})
