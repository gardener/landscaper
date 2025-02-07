package manifestdeployer

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Manifest Deployer Installer Test Suite")
}

var _ = Describe("Manifest Deployer Installer", func() {

	const id = "test-g23tp"

	It("should install the manifest deployer", func() {
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

		err = InstallManifestDeployer(ctx, values)
		Expect(err).ToNot(HaveOccurred())
	})

	XIt("should uninstall the manifest deployer", func() {
		ctx := context.Background()

		values := &Values{
			Key: NewKeyFromID(id),
		}

		err := UninstallManifestDeployer(ctx, values)
		Expect(err).ToNot(HaveOccurred())
	})

})
