package manifestdeployer

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
	RunSpecs(t, "Manifest Deployer Installer Test Suite")
}

var _ = Describe("Manifest Deployer Installer", func() {

	const id = "test-g23tp"

	newHostClient := func() (client.Client, error) {
		cfg, err := config.GetConfig()
		if err != nil {
			return nil, fmt.Errorf("unable to load kubeconfig for host cluster of manifest deployer: %v\n", err)
		}
		hostClient, err := client.New(cfg, client.Options{})
		if err != nil {
			return nil, fmt.Errorf("unable to create kubernetes client for host cluster of manifest deployer: %v\n", err)
		}
		return hostClient, nil
	}

	It("should install the manifest deployer", func() {
		ctx := context.Background()

		kubeconfig, err := os.ReadFile(os.Getenv("KUBECONFIG"))
		Expect(err).ToNot(HaveOccurred())

		values := &Values{
			Instance: id,
			Version:  "v0.127.0",
			LandscaperClusterKubeconfig: &KubeconfigValues{
				Kubeconfig: string(kubeconfig),
			},
			Image: ImageValues{
				Repository: "europe-docker.pkg.dev/sap-gcp-cp-k8s-stable-hub/landscaper/github.com/gardener/landscaper/manifest-deployer/images/manifest-deployer-controller",
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

		err = InstallManifestDeployer(ctx, hostCl, values)
		Expect(err).ToNot(HaveOccurred())
	})

	XIt("should uninstall the manifest deployer", func() {
		ctx := context.Background()

		values := &Values{
			Instance: id,
		}

		hostCl, err := newHostClient()
		Expect(err).ToNot(HaveOccurred())

		err = UninstallManifestDeployer(ctx, hostCl, values)
		Expect(err).ToNot(HaveOccurred())
	})

})
