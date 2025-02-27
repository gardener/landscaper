package rbac

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"testing"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Landscaper RBAC Installer Test Suite")
}

var _ = Describe("Landscaper RBAC Installer", func() {

	const id = "test-rr8fq"

	newResourceClient := func() (client.Client, error) {
		cfg, err := config.GetConfig()
		if err != nil {
			return nil, fmt.Errorf("unable to load kubeconfig for resource cluster: %v\n", err)
		}
		hostClient, err := client.New(cfg, client.Options{})
		if err != nil {
			return nil, fmt.Errorf("unable to create kubernetes client for resource cluster: %v\n", err)
		}
		return hostClient, nil
	}

	It("should install the landscaper rbac resources", func() {
		ctx := context.Background()

		values := &Values{
			Key:            NewKeyFromID(id),
			Version:        "v0.127.0",
			ServiceAccount: &ServiceAccountValues{Create: true},
		}

		resourceCl, err := newResourceClient()
		Expect(err).ToNot(HaveOccurred())

		err = InstallLandscaperRBACResources(ctx, resourceCl, values)
		Expect(err).ToNot(HaveOccurred())
	})

	XIt("should uninstall the landscaper rbac resources", func() {
		ctx := context.Background()

		values := &Values{
			Key: NewKeyFromID(id),
		}

		resourceCl, err := newResourceClient()
		Expect(err).ToNot(HaveOccurred())

		err = UninstallLandscaperRBACResources(ctx, resourceCl, values)
		Expect(err).ToNot(HaveOccurred())
	})

})
